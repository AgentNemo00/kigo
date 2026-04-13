package config

import (
	"context"
	"fmt"

	"github.com/AgentNemo00/sca-instruments/api/router"
	"github.com/AgentNemo00/sca-instruments/errors"
	"github.com/AgentNemo00/sca-instruments/log"
	"github.com/AgentNemo00/kigo-core/util"
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
