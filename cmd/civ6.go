package cmd

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/clbx/juicebot/util"
)

// Civ6Payload is the body Civ 6 "Play By Cloud" POSTs when it becomes a
// player's turn. It uses the legacy IFTTT-style value1/value2/value3 fields:
//
//	value1 = game name, value2 = player name, value3 = turn number (as a string)
//
// Pointers are used so missing fields can be distinguished from empty strings.
type Civ6Payload struct {
	Value1 *string `json:"value1"`
	Value2 *string `json:"value2"`
	Value3 *string `json:"value3"`
}

// Civ6WebhookHandler returns an http.HandlerFunc that receives Civ 6 turn
// notifications, logs each turn to the DB, and announces it in the configured
// Discord channel (pinging the mapped player). Requests whose {secret} path
// segment doesn't match the configured secret are rejected with 404.
func Civ6WebhookHandler(s *discordgo.Session, db *sql.DB, config *util.JuiceBotConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Authenticate via the secret embedded in the URL path.
		secret := r.PathValue("secret")
		if config.Civ6.WebhookSecret == "" ||
			subtle.ConstantTimeCompare([]byte(secret), []byte(config.Civ6.WebhookSecret)) != 1 {
			http.NotFound(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("civ6 webhook: failed to read body: %v", err)
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		var payload Civ6Payload
		turnNumber := 0
		if err := json.Unmarshal(body, &payload); err == nil &&
			payload.Value1 != nil && payload.Value2 != nil && payload.Value3 != nil {
			if n, convErr := strconv.Atoi(*payload.Value3); convErr == nil {
				turnNumber = n
			} else {
				payload.Value3 = nil // mark as malformed below
			}
		}

		// Anything that isn't the expected 3-field numeric shape: dump the raw
		// JSON into the channel so a human can see what Civ actually sent.
		if payload.Value1 == nil || payload.Value2 == nil || payload.Value3 == nil {
			dump := fmt.Sprintf("Received unexpected Civ6 webhook payload:\n```json\n%s\n```", string(body))
			if _, sendErr := s.ChannelMessageSend(config.Civ6.ChannelID, dump); sendErr != nil {
				log.Printf("civ6 webhook: failed to post malformed-payload dump: %v", sendErr)
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		gameName := *payload.Value1
		playerName := *payload.Value2

		if err := util.AddCivTurn(db, util.CivTurnEntry{
			GameName:   gameName,
			PlayerName: playerName,
			TurnNumber: turnNumber,
		}); err != nil {
			log.Printf("civ6 webhook: failed to log turn: %v", err)
			// Still announce even if the DB write failed.
		}

		// Build the announcement, pinging the mapped Discord user if we know them.
		var allowedMentions *discordgo.MessageAllowedMentions
		playerRef := playerName
		if discordID, ok := config.Civ6.Players[playerName]; ok && discordID != "" {
			playerRef = fmt.Sprintf("<@%s>", discordID)
			allowedMentions = &discordgo.MessageAllowedMentions{
				Users: []string{discordID},
			}
		} else {
			log.Printf("civ6 webhook: no Discord user mapped for player %q, falling back to game username", playerName)
		}

		content := fmt.Sprintf("%s, it's your turn!\nGame: %s · Turn: %d", playerRef, gameName, turnNumber)
		if _, err := s.ChannelMessageSendComplex(config.Civ6.ChannelID, &discordgo.MessageSend{
			Content:         content,
			AllowedMentions: allowedMentions,
		}); err != nil {
			log.Printf("civ6 webhook: failed to send turn announcement: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}
}
