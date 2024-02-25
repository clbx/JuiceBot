package cmd

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var dogId = ""

//which guild is which user being dogged in
var dogging = make(map[string]string)

var DogCommand = &discordgo.ApplicationCommand{
	Name:        "dog",
	Description: "Homophobia is bad",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "User to remind",
			Required:    false,
		},
	},
}

func DogAction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))

	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	if len(options) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Dogging Disabled",
			},
		})
		dogId = ""
	}

	if opt, ok := optionMap["user"]; ok {

		dogging[i.GuildID] = opt.UserValue(nil).ID

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Now Dogging: <@%s>", dogId),
			},
		})
	}

}

func DogMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == dogging[m.GuildID] {
		err := s.MessageReactionAdd(m.ChannelID, m.ID, "homophobic:985705480791937066")
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error Encountered: %s", err))
		}
	}

	emojis := m.GetCustomEmojis()
	for i := 0; i < len(emojis); i++ {
		fmt.Printf("%s\n", emojis[i].ID)
	}

}
