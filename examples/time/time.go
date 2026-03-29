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
	"github.com/beevik/ntp"
)

const(
	
	TimeLocationChange = "TimeLocationChange"

	TimeChangeLocationOrder = "TimeChangeLocationOrder"
)

type TimeModule struct {
	router.Config
	TimeZone string
	Format   string
	NTPServer string
	location *time.Location
	restart bool
	handler *router.Handler
	time time.Time
}

func (t *TimeModule) Default() {
	if t.TimeZone == "" {
		t.TimeZone = "Europe/Berlin"
	}
	if t.Format == "" {
		t.Format = "15:04"
	}
}

func (t *TimeModule) OnStartUp(ctx *context.Context, payload core.PayloadStartUp) (*core.RespStartUp, error) {
	loc, err := time.LoadLocation(t.TimeZone)
	if err != nil {
		log.Ctx(ctx).Err(err)
		return nil, err
	}
	t.location = loc
	ntpTime, err := ntp.Time(t.NTPServer)
	if err != nil {
		log.Ctx(ctx).Err(err)
		return nil, err
	}
	t.time = ntpTime.In(t.location)
	return &core.RespStartUp{
		NotificationsOn: []string{
			core.NotificationStartUp,
			core.NotificationShutdown,
			core.NotificationReboot,
			core.NotificationUpdate,
			core.NotificationRender,
			TimeChangeLocationOrder,
		},
		NotificationsSend: []string{
			TimeLocationChange,
		},
		CallingDuration: time.Minute,
	}, nil
}

func (t *TimeModule) OnShutdown(ctx *context.Context) error {
	t.restart = false
	err := t.handler.Stop(ctx.Request().Context())
	if err != nil {
		log.Ctx(ctx).Err(err)
		return err
	}
	return nil
}

func (t *TimeModule) OnReboot(ctx *context.Context) (*core.RespReboot, error) {
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

func (t *TimeModule) OnUpdate(ctx *context.Context, payload core.PayloadUpdate) (*core.RespUpdate, error) {
	if len(payload.Payload) == 0 {
		return &core.RespUpdate{
			Duration: 0,
			NotificationsSend: []string{},
		}, nil
	}

	newLocStr, ok := payload.Payload[TimeChangeLocationOrder].(string)
	if !ok {
		err := fmt.Errorf("invalid payload for %s", TimeChangeLocationOrder)
		log.Ctx(ctx).Err(err)
		return nil, err
	}
	newLoc, err := time.LoadLocation(newLocStr)
	if err != nil {
		log.Ctx(ctx).Err(err)
		return nil, err
	}
	t.location = newLoc

	return &core.RespUpdate{
		Duration: 0,
		NotificationsSend: []string{TimeLocationChange},
	}, nil
}

func (t *TimeModule) OnRender(ctx *context.Context, payload core.PayloadRender) (*core.RespRender, error) {
	ntpTime, err := ntp.Time(t.NTPServer)
	if err != nil {
		log.Ctx(ctx).Err(err)
		return nil, err
	}
	localtime := ntpTime.In(t.location).Format(t.Format)
	return &core.RespRender{
		Object: localtime,
	}, nil
}

func main() {
	module := &TimeModule{}
	err := configuration.ByEnv(module)
	if err != nil {
		log.Ctx(c.Background()).Err(err)
		return
	}
	route := core.WrapModuleWithRoute(module)

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
