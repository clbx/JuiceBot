package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/clbx/juicebot/util"
)

var NameHistoryCommand = &discordgo.ApplicationCommand{
	Name:        "namehistory",
	Description: "Gets the nickname history of a user",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "user to lookup",
			Required:    true,
		},
	},
}

func NameHistoryHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate, config *util.JuiceBotConfig, db *sql.DB) {
	log.Printf("User %s in guild %s changed their name to %s", m.User.ID, m.GuildID, m.Nick)
	if !slices.Contains(config.NameHistoryGuilds, m.GuildID) {
		return
	}
	err := util.AddNameEntry(db, util.NameDBEntry{
		GuildID:        m.GuildID,
		UserID:         m.User.ID,
		NewDisplayName: m.Nick,
	})

	if err != nil {
		log.Printf("Failed to write name change to the DB %w", err)
	}

	message := fmt.Sprintf("<@%s> has a new name!", m.User.ID)
	_, err = s.ChannelMessageSend("1422037671563235489", message)
	if err != nil {
		log.Printf("Failed to send name change announcement: %v", err)
	}

}

func NameHistoryAction(s *discordgo.Session, i *discordgo.InteractionCreate, config *util.JuiceBotConfig, db *sql.DB) {
	if !slices.Contains(config.NameHistoryGuilds, i.GuildID) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Name history is not enabled on this server",
			},
		})
		return
	}

	options := i.ApplicationCommandData().Options
	userID := options[0].UserValue(s).ID

	history, err := util.GetNameHistory(db, i.GuildID, userID)
	if err != nil {
		log.Printf("Failed to get name history: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to retrieve name history",
			},
		})
		return
	}

	if len(history) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No name history found for this user",
			},
		})
		return
	}

	// Build a formatted table
	var response string
	response = "```\n"
	response += "Name History\n"
	response += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	response += "Nickname                      | Changed At\n"
	response += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"

	for _, entry := range history {
		nickname := entry.NewDisplayName
		if nickname == "" {
			nickname = "(none)"
		}
		if len(nickname) > 29 {
			nickname = nickname[:29]
		}
		response += fmt.Sprintf("%-30s| %s\n", nickname, entry.ChangedAt)
	}

	response += "```"

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
		},
	})
}
