package util

import (
	"os"

	"gopkg.in/yaml.v2"
)

type JuiceBotConfig struct {
	Debug             bool     `yaml:"debug"`
	NameHistoryGuilds []string `yaml:"nameHistoryGuilds"`
	DogConfig         struct {
		DogEmote string `yaml:"dogEmote"`
	} `yaml:"dogConfig"`
	CalloutConfig struct {
		CalloutGuilds   []string `yaml:"calloutGuilds"`
		CalloutMessages []string `yaml:"calloutMessages"`
		CalloutChance   int      `yaml:"calloutChance"`
	} `yaml:"calloutConfig"`
	DB struct {
		Path string `yaml:"path"`
	} `yaml:"db"`
	Games struct {
		Channels []struct {
			GuildID   string `yaml:"guildid"`
			ChannelID string `yaml:"channelid"`
		} `yaml:"channels"`
	} `yaml:"games"`
	Civ6 struct {
		GuildID       string            `yaml:"guildid"`
		ChannelID     string            `yaml:"channelid"`
		WebhookSecret string            `yaml:"webhookSecret"`
		ListenAddress string            `yaml:"listenAddress"` // optional, default ":8080"
		Players       map[string]string `yaml:"players"`       // playerName -> discord user ID
	} `yaml:"civ6"`
}

func NewJuiceBotConfig(configPath string) *JuiceBotConfig {
	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	var config JuiceBotConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	return &config
}
