package cmd

import "github.com/bwmarrin/discordgo"

var PingCommand = &discordgo.ApplicationCommand{
	Name:        "ping",
	Description: "pong",
}

func PingAction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "pong!",
		},
	})
}
