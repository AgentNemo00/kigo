package config

import (
	"fmt"
	"sync"

	"github.com/AgentNemo00/sca-instruments/api/router"
	"github.com/AgentNemo00/sca-instruments/errors"
)

type Config struct {
	router.Config
	PubSubUrl string

	Modules []Module

	mu sync.RWMutex
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
		c.Modules = []Module{
			{Name: "Module1", Path: "./modules/welcomeKiGo"},
		}
	}
}

func (c *Config) GetModule(name string) (*Module, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for index, module := range c.Modules {
		if module.Name == name {
			return &module, index
		}
	}
	return nil, -1
}

func (c *Config) CreateModule(name string) (*Module, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	module, _ := c.GetModule(name)
	if module != nil {
		return nil, errors.New(fmt.Errorf("A module with this name already exits"))
	}
	newModule := Module{
		Name: name
	}
	c.Modules = append(c.Modules, newModule)
	return &newModule
}

