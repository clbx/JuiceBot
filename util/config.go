package util

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type JuiceBotConfig struct {
	Debug     bool `yaml:"debug"`
	DogConfig struct {
		DogEmote string `yaml:"dogEmote"`
	} `yaml:"dogConfig"`
	CalloutConfig struct {
		CalloutGuilds   []string `yaml:"calloutGuilds"`
		CalloutMessages []string `yaml:"calloutMessages"`
		CalloutChance   int      `yaml:"calloutChance"`
	} `yaml:"calloutConfig"`
}

func NewJuiceBotConfig(configPath string) *JuiceBotConfig {
	data, err := ioutil.ReadFile(configPath)
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
