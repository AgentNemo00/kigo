package main

import (
	"context"

	"github.com/AgentNemo00/kigo-ui/handler"
	"github.com/AgentNemo00/kigo-ui/window"
	"github.com/AgentNemo00/sca-instruments/configuration"
	"github.com/AgentNemo00/sca-instruments/containerization"
	"github.com/AgentNemo00/sca-instruments/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	w, err := window.NewWindow(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err)
		return
	}
	
	c := &handler.Config{}
	err = configuration.ByEnv(c)
	if err != nil {
		log.Ctx(ctx).Err(err)
		return
	}

	app, err := handler.NewHandler(c, w)
	if err != nil {
		log.Ctx(ctx).Err(err)
		return
	}
	
	containerization.Callback(func ()  {
		app.Stop(ctx)
		w.Stop()
		cancel()
	})
	
	go containerization.Interrupt(func() {})
	err = app.Start(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err)
		return
	}
	<- ctx.Done()
}

