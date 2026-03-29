package main

import (
	"fmt"
	"time"
	c "context"
	"github.com/kataras/iris/v12/context"

	"github.com/AgentNemo00/sca-instruments/configuration"
	"github.com/AgentNemo00/sca-instruments/log"
	"github.com/AgentNemo00/sca-instruments/api/router"
	"github.com/AgentNemo00/sca-instruments/containerization"
	"github.com/agentnemo00/kigo/core"
)

const(
	TextTextChange = "TextTextChange"

	TextChangeTextOrder = "TextChangeTextOrder"
)

type TextModule struct {
	router.Config
	Value string
	restart bool
	handler *router.Handler
}

func (t *TextModule) Default() {
	if t.Value == "" {
		t.Value = "Welcome to Kigo!"
	}
}

func (t *TextModule) OnStartUp(ctx *context.Context, payload core.PayloadStartUp) (core.RespStartUp, error) {
	return core.RespStartUp{
		NotificationsOn: []string{
			core.NotificationStartUp,
			core.NotifcationShutdown,
			core.NotificationReboot,
			core.NotificationUpdate,
			core.NotificationRender,
			TextChangeTextOrder,
		},
		NotificationsSend: []string{
			TextTextChange,
		},
		CallingDuration: time.Minute,
	}, nil
}

func (t *TextModule) OnShutdown(ctx *context.Context) error {
	t.restart = false
	err := t.handler.Stop(ctx.Request().Context())
	if err != nil {
		log.Ctx(ctx).Err(err)
		return err
	}
	return nil
}

func (t *TextModule) OnReboot(ctx *context.Context) (*core.RespReboot, error) {
	t.restart = true
	err := t.handler.Stop(ctx.Request().Context())
	if err != nil {
		log.Ctx(ctx).Err(err)
		return nil, err
	}
	return &core.RespReboot{
		Duration: 2 * time.Second,
	}, nil
}

func (t *TextModule) OnUpdate(ctx *context.Context, payload core.PayloadUpdate) (*core.RespUpdate, error) {
	if len(payload.Payload) == 0 {
		return &core.RespUpdate{
			Duration: 0,
			NotificationsSend: []string{},
		}, nil
	}

	newText, ok := payload.Payload[TextChangeTextOrder].(string)
	if !ok {
		err := fmt.Errorf("invalid payload for %s", TextChangeTextOrder)
		log.Ctx(ctx).Err(err)
		return nil, err
	}
	t.Value = newText
	return &core.RespUpdate{
		Duration: 0,
		NotificationsSend: []string{TextTextChange},
	}, nil
}

func (t *TextModule) OnRender(ctx *context.Context, payload core.PayloadRender) (*core.RespRender, error) {
	return &core.RespRender{
		Object: t.Value,
	}, nil
}

func main() {
	module := &TextModule{}
	err := configuration.ByEnv(module)
	if err != nil {
		log.Ctx(c.Background()).Err(err)
		return
	}
	route:= core.WrapModuleWithRoute(module)

	interrupt := false
	containerization.Callback(func ()  {
		interrupt = true
	})

	for module.restart == true && interrupt == false {
		go containerization.Interrupt(func() {})

		handler, err := core.BuildHandler(&module.Config, route)
		if err != nil {
			log.Ctx(c.Background()).Err(err)
			return
		}

		err = handler.Start()
		if err != nil {
			fmt.Println("Error occurred while starting the handler:", err)
			return
		}
	}

}
