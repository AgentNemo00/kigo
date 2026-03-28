package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/agentnemo00/kigo/module"
	"github.com/kataras/iris/v12/context"

	"github.com/AgentNemo00/sca-instruments/api/errors"
	"github.com/AgentNemo00/sca-instruments/api/response"
	"github.com/AgentNemo00/sca-instruments/api/router"
	"github.com/AgentNemo00/sca-instruments/api/router/routen"
	"github.com/AgentNemo00/sca-instruments/api/validation"
	"github.com/AgentNemo00/sca-instruments/configuration"
	"github.com/AgentNemo00/sca-instruments/containerization"
	"github.com/AgentNemo00/sca-instruments/log"
)

type TextModule struct {
	router.Config
	Text 	string
	Version string
}

func (t *TextModule) Default() {
	if t.Name == "" {
		t.Name = "KigoTextModule"
	}
	if t.Text == "" {
		t.Text = "Welcome to Kigo"
	}
	if t.Version == "" {
		t.Version = "1.0.0"
	}
}

func Config() (*TextModule, error) {
	config := &TextModule{
		Config: router.Config{
			Port: 10001,
		},
	}
	return config, configuration.ByEnv(config)
}

func InjectNotificationHandler(handler *router.Handler, config *TextModule, restart *bool) routen.Route {
	return routen.NewParams(
		routen.Base{
			Method: http.MethodPost,
			Path: "/notification/{value:string}",
			Name: "notification",
		},
		func(ctx *context.Context, params ...string) {
			fmt.Println("Notification")
			switch params[0] {
			case module.NotificationStartUp:
				var payload module.PayloadStartUp
				err := ctx.ReadJSON(&payload)
				if err != nil {
					validation.NewUnmarshalError().Parse(ctx)
					log.Ctx(ctx).Err(err)
					return
				}
				response.Parse[module.RespStartUp](ctx, http.StatusOK, module.RespStartUp{
					NotificationsOn: []string{
						config.Name,
						module.NotificationStartUp,
						module.NotifcationShutdown,
						module.NotificationReboot,
					},
				},
				)
			case module.NotifcationShutdown:
				*restart = false
				err := handler.Stop(ctx.Request().Context())
				if err != nil {
					log.Ctx(ctx).Err(err)
					return
				}
			case module.NotificationReboot:
				time.AfterFunc(time.Second, func() {
					err := handler.Stop(ctx.Request().Context())
					if err != nil {
						log.Ctx(ctx).Err(err)
						return
					}
				})
				response.Parse[module.RespReboot](ctx, http.StatusOK, 
					module.RespReboot{
						Duation: 5 * time.Second,
				})
			case module.NotificationUpdate:
				response.Parse[module.RespUpdate](ctx, http.StatusOK, 
					module.RespUpdate{
						Duration: time.Second,
				})
			case module.NotificationRender:
				var payload module.PayloadRender
				err := ctx.ReadJSON(&payload)
				if err != nil {
					validation.NewUnmarshalError().Parse(ctx)
					log.Ctx(ctx).Err(err)
					return
				}
				// TODO render
				response.Parse[module.RespRender](ctx, http.StatusOK, 
					module.RespRender{
						PositionX: 0,
						PositionY: 0,
						Object: nil,
					})
			default:
				errors.NotFound(ctx, fmt.Sprintf("NotificationType invalid: %s", params[0]))
				return
			}
		},
		"value",
	)
}

func ApplicatonHandler(config *TextModule, restart *bool) (*router.Handler, error) {
		handler := router.NewHandler(&config.Config)

		route := InjectNotificationHandler(handler, config, restart)
		group := router.NewGroup(config.Name, route)
	
		err := handler.Build(router.Base{
			Version:     *router.NewVersion(config.Version),
			Groups:      []router.Group{group},
		})
		if err != nil {
			return nil, err
		}
		return handler, nil
}

func main() {
	config, err := Config() 
	if err != nil {
		fmt.Println("Error occurred while loading configuration:", err)
		return
	}

	restart := true
	interrupt := false
	containerization.Callback(func ()  {
		interrupt = true
	})

	for restart == true && interrupt == false {
		go containerization.Interrupt(func() {})

		handler, err := ApplicatonHandler(config, &restart)
		if err != nil {
			fmt.Println("Error occurred while creating the handler:", err)
			return
		}

		err = handler.Start()
		if err != nil {
			fmt.Println("Error occurred while starting the handler:", err)
			return
		}
	}
	fmt.Println("Shutdown")
}

