package util

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type JuiceBotConfig struct {
	CalloutConfig struct {
		CalloutGuilds    []string
		CalloutMessages  []string
		CalloutFrequency int
		CalloutVariance  int
	}
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
