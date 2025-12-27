package cmd

import (
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/clbx/juicebot/util"
)

func NameHistoryHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate, config *util.JuiceBotConfig) {
	// Don't run if the guild is not in the list of guilds where name tracking is enabeld.
	if !slices.Contains(config.NameHistoryGuilds, m.GuildID) {
		return
	}

}
