package cmd

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var VersionCommand = &discordgo.ApplicationCommand{
	Name:        "version",
	Description: "Prints out version information",
}

func VersionAction(s *discordgo.Session, i *discordgo.InteractionCreate) {

	versionInfo := fmt.Sprintf("JuiceBot Version: ")
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: versionInfo,
		},
	})
}
