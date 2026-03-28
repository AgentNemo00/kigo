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
	"github.com/AgentNemo00/sca-instruments/log"
)

var (
	handler *router.Handler = nil
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
}

func main() {
	config := &TextModule{}
	err := configuration.ByEnv(config)
	if err != nil {
		fmt.Println("Error occurred while loading configuration:", err)
		return
	}

	notificationRoute := routen.NewParams(
		routen.Base{
			Method: http.MethodGet,
			Path: "/notification/{value:string}",
			Name: "notification",
		},
		func(ctx *context.Context, params ...string) {
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
						Duation: 10 * time.Second,
				})
			case module.NotificationUpdate:
				response.Parse[module.RespUpdate](ctx, http.StatusOK, 
					module.RespUpdate{
						Duration: 2 * time.Second,
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
				errors.NotFound(ctx, "NotificationType invalid")
				return
			}
		},
		"value",
	)

	group := router.NewGroup(config.Name, notificationRoute)
	
	base := router.Base{
		Version:     *router.NewVersion(config.Version),
		Groups:      []router.Group{group},
	}

	handler := router.NewHandler(&config.Config)

	err = handler.Build(base)
	if err != nil {
		fmt.Println("Error occurred while building the handler:", err)
		return
	}

	err = handler.Start()
	if err != nil {
		fmt.Println("Error occurred while starting the handler:", err)
		return
	}
}
