package cmd

import (
	"database/sql"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/clbx/juicebot/util"
)

var AddGameCommand = &discordgo.ApplicationCommand{
	Name:        "addgame",
	Description: "Adds a game to the game list ",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "game",
			Description: "game to add to the list",
			Required:    true,
		},
	},
}

var RemoveGameCommand = &discordgo.ApplicationCommand{
	Name:        "removegame",
	Description: "Removes a game from the game list ",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "game",
			Description: "game to remove from the list",
			Required:    true,
		},
	},
}

func AddGameAction(s *discordgo.Session, i *discordgo.InteractionCreate, config *util.JuiceBotConfig, db *sql.DB) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))

	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var responseMsg string

	if opt, ok := optionMap["game"]; ok {
		exists, _ := gameExistsInDB(opt.StringValue(), db)
		if exists {
			responseMsg = "Game already exists in list"
		} else {

			//Add the game to the db
			addGameSQL := `INSERT INTO games (name) VALUES (?)`
			_, err := db.Exec(addGameSQL, opt.StringValue())
			if err != nil {
				responseMsg = fmt.Sprintf("error adding game to db: %s", err)
			} else {
				responseMsg = fmt.Sprintf("Added %s to game list", opt.StringValue())
			}

		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: responseMsg,
			},
		})

	}
}

func RemoveGameAction(s *discordgo.Session, i *discordgo.InteractionCreate, config *util.JuiceBotConfig, db *sql.DB) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))

	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var responseMsg string

	if opt, ok := optionMap["game"]; ok {
		exists, _ := gameExistsInDB(opt.StringValue(), db)
		if exists {
			//Add the game to the db
			removeGameSQL := `DELETE FROM games WHERE name=?`
			_, err := db.Exec(removeGameSQL, opt.StringValue())
			if err != nil {
				responseMsg = fmt.Sprintf("error adding removing from db: %s", err)
			} else {
				responseMsg = fmt.Sprintf("Remove %s from game list", opt.StringValue())
			}

		} else {
			responseMsg = "Game does not exist in list"
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: responseMsg,
			},
		})

	}
}

func gameExistsInDB(gameName string, db *sql.DB) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM games WHERE name=? LIMIT 1)", gameName).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
