package cmd

import (
	"github.com/bwmarrin/discordgo"
)

var CalloutCommand = &discordgo.ApplicationCommand{
	Name: "callout",
}

func CalloutAction(s *discordgo.Session, i *discordgo.InteractionCreate) {

}
