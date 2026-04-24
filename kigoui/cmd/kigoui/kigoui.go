package main

import (
	"context"

	"github.com/AgentNemo00/sca-instruments/configuration"
	"github.com/AgentNemo00/sca-instruments/containerization"
	"github.com/AgentNemo00/sca-instruments/log"
	"github.com/AgentNemo00/kigo-ui/handler"
)

func main() {
	c := &handler.Config{}
	err := configuration.ByEnv(c)
	if err != nil {
		log.Ctx(context.Background()).Err(err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())

	app, err := handler.NewHandler(c)
	if err != nil {
		log.Ctx(context.Background()).Err(err)
		return
	}
	
	containerization.Callback(func ()  {
		app.Stop(ctx)
		cancel()
	})
	
	go containerization.Interrupt(func() {})
	err = app.Start(ctx)
	if err != nil {
		log.Ctx(context.Background()).Err(err)
		return
	}
	<- ctx.Done()
}

