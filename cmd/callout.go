package cmd

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/clbx/juicebot/util"
)

func CalloutHandler(s *discordgo.Session, m *discordgo.MessageCreate, config *util.JuiceBotConfig) {

	var contains bool

	for _, item := range config.CalloutConfig.CalloutGuilds {
		if item == m.GuildID {
			contains = true
			break
		}
	}

	if !contains {
		return
	}

	rand.Seed(time.Now().UnixNano())
	if rand.Intn(config.CalloutConfig.CalloutChance) == 0 {

		content := config.CalloutConfig.CalloutMessages[rand.Intn(len(config.CalloutConfig.CalloutMessages))]

		msg := &discordgo.MessageSend{
			Content: content,
			Reference: &discordgo.MessageReference{
				MessageID: m.ID,
				ChannelID: m.ChannelID,
			},
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{},
			},
		}

		_, err := s.ChannelMessageSendComplex(m.ChannelID, msg)

		if err != nil {
			fmt.Println("Error sending callout message: ", err)
		}
	}
}
