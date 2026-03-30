package config

import "github.com/AgentNemo00/sca-instruments/api/router"

type Config struct {
	router.Config
	Modules []Module
	PubSubUrl string
	// TODO: output to draw to
}

func (c *Config) Default() {
	if c.Port == 0 {
		c.Port = 10001
	}
	if c.Name == "" {
		c.Name = "kigo"
	}
	if c.Version == "" {
		c.Version = "1.0.0"
	}
	if len(c.Modules) == 0 {
		c.Modules = []Module{
			{Name: "Module1", Path: "./modules/welcomeKiGo"},
		}
	}
}

func (c *Config) GetModule(name string) (*Module, int) {
	for index, module := range c.Modules {
		if module.Name == name {
			return &module, index
		}
	}
	return nil, -1
}

