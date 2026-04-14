package config

import (
	"github.com/AgentNemo00/sca-instruments/api/router"
)

type Config struct {
	router.Config
	PubSubUrl string

	Modules []Module

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
	if len(c.Modules) == 0 {
		c.Modules = nil // TODO: append welcome module
	}
}
