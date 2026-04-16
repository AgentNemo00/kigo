package config

import (
	"github.com/AgentNemo00/sca-instruments/api/router"
)

type Config struct {
	router.Config
	
	PubSubUrl string

	KiGoUIID string
	// TODO: output to draw to
}

func (c *Config) Default() {
	if c.Port == 0 {
		c.Port = 10001
	}
	if c.Name == "" {
		c.Name = "KigoMainService"
	}
	if c.Version == "" {
		c.Version = "1.0.0"
	}
	if c.PubSubUrl == "" {
		c.PubSubUrl = "nats://127.0.0.1:4222"
	}
	if c.KiGoUIID == "" {
		c.KiGoUIID = "kigoui://127.0.0.1:53740"
	}
}
