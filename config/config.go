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
		c.Modules = []Module{
			{Name: "Module1", Path: "./modules/welcomeKiGo"},
		}
	}
}

// Does a check on modules data and removes if it does not fit
func (c *Config) CheckModules(ctx context.Context) error {
	newModuleList := make([]Module, len(c.Modules))
	for index, module := range c.Modules {
		// check rules
		if module.Path != "" {
			if !util.Exists(module.Path) {
				log.Ctx(ctx).Debug("module %d has an invalid path", index)
				continue
			}
		}

		newModuleList = append(newModuleList, module)
	}
	// if modules got removed and there are now empty
	if len(c.Modules) != len(newModuleList) && len(newModuleList) == 0 {
		return errors.New(fmt.Errorf("no modules left after checks"))
	}
	c.Modules = newModuleList
	return nil
}
