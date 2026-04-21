package config

import (
	"github.com/AgentNemo00/kigo-core/ui"
	"github.com/AgentNemo00/sca-instruments/api/router"
)

type Config struct {
	router.Config
	
	PubSubUrl 			string
	KiGoUI        		string

	SupportedFormats   	[]string
	SupportedChannels  	[]string
	SupportedFPS       	int
	
	Database			string
}

func (c *Config) Default() {
	if c.Port == 0 {
		c.Port = 10001
	}
	if c.Name == "" {
		c.Name = "KiGo"
	}
	if c.Version == "" {
		c.Version = "1.0.0"
	}
	if c.PubSubUrl == "" {
		c.PubSubUrl = "nats://127.0.0.1:4222"
	}
	if c.KiGoUI == "" {
		c.KiGoUI = "KiGoUI"
	}
	if c.SupportedFPS == 0 {
		c.SupportedFPS = 24
	}
	if len(c.SupportedFormats) == 0 {
		c.SupportedFormats = []string{ui.RAW}
	}
	if len(c.SupportedChannels) == 0 {
		c.SupportedChannels = []string{ui.PubSub}
	}
	if c.Database == "" {
		c.Database = "kigo.db"
	}
}
