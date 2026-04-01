package main

import (
	"context"
	"github.com/AgentNemo00/kigo/service"

	"github.com/AgentNemo00/sca-instruments/configuration"
	"github.com/AgentNemo00/sca-instruments/containerization"
	"github.com/AgentNemo00/sca-instruments/log"
	"github.com/AgentNemo00/kigo/config"
)

func main() {
	c := &config.Config{}
	err := configuration.ByEnv(c)
	if err != nil {
		log.Ctx(context.Background()).Err(err)
		return
	}
	if len(c.Modules) == 0 {
		log.Ctx(context.Background()).Error("no modules detected")
	}
	ctx, cancel := context.WithCancel(context.Background())
	err = c.CheckModules(ctx)
	if err != nil {
		log.Ctx(context.Background()).Err(err)
		return
	}

	app, err := service.NewService(c)
	if err != nil {
		log.Ctx(context.Background()).Err(err)
		return
	}
	
	containerization.Callback(func ()  {
		err := app.Stop(ctx)
		if err != nil {
			log.Ctx(context.Background()).Err(err)
		}
		cancel()
	})
	
	go containerization.Interrupt(func() {})
	err = app.Start(ctx)
	if err != nil {
		log.Ctx(context.Background()).Err(err)
		return
	}	
}

