package core

import (
	"net/http"

	"github.com/agentnemo00/kigo/core"
	"github.com/kataras/iris/v12/context"

	"github.com/AgentNemo00/sca-instruments/api/errors"
	"github.com/AgentNemo00/sca-instruments/api/response"
	"github.com/AgentNemo00/sca-instruments/api/router"
	"github.com/AgentNemo00/sca-instruments/api/router/routen"
	"github.com/AgentNemo00/sca-instruments/api/validation"
	"github.com/AgentNemo00/sca-instruments/log"
)

type ModuleInterface interface {
	OnStartUp(ctx *context.Context,payload PayloadStartUp) (RespStartUp, error)
	OnShutdown(ctx *context.Context) error
	OnReboot(ctx *context.Context) (RespReboot, error)
	OnUpdate(ctx *context.Context, payload PayloadUpdate) (RespUpdate, error)
	OnRender(ctx *context.Context, payload PayloadRender) (RespRender, error)
}

func WrapModuleWithRoute(module ModuleInterface) routen.Route {
	return routen.NewParams(
		routen.Base{
			Method: http.MethodPost,
			Path: "/notification/{notification:string}",
			Name: "notification",
		},
		func(ctx *context.Context, params ...string) {
			switch params[0] {
				case core.NotificationStartUp:
					var payload core.PayloadStartUp
					err := ctx.ReadJSON(&payload)
					if err != nil {
						validation.NewUnmarshalError().Parse(ctx)
						log.Ctx(ctx).Err(err)
						return
					}
					resp, err := module.OnStartUp(ctx, payload)
					if err != nil {
						errors.Internal(ctx, err)
						log.Ctx(ctx).Err(err)
						return
					}
					response.Parse[core.RespStartUp](ctx, http.StatusOK, resp)
				case core.NotificationShutdown:
					err := module.OnShutdown()
					if err != nil {
						errors.Internal(ctx, err)
						log.Ctx(ctx).Err(err)
						return
					}
				case core.NotificationReboot:
					resp, err := module.OnReboot(ctx)
					if err != nil {
						errors.Internal(ctx, err)
						log.Ctx(ctx).Err(err)
						return
					}
					response.Parse[core.RespReboot](ctx, http.StatusOK, resp)
				case core.NotifiNotificationUpdate:
					var payload core.PayloadUpdate
					err := ctx.ReadJSON(&payload)
					if err != nil {
						validation.NewUnmarshalError().Parse(ctx)
						log.Ctx(ctx).Err(err)
						return
					}
					resp, err := module.OnUpdate(ctx, payload)
					if err != nil {
						errors.Internal(ctx, err)
						log.Ctx(ctx).Err(err)
						return
					}
					response.Parse[core.module.RespUpdate](ctx, http.StatusOK, resp)
				case core.NotificationRender:
					var payload core.PayloadRender
					err := ctx.ReadJSON(&payload)
					if err != nil {
						validation.NewUnmarshalError().Parse(ctx)
						log.Ctx(ctx).Err(err)
						return
					}
					resp, err := module.OnRender(ctx, payload)
					if err != nil {
						errors.Internal(ctx, err)
						log.Ctx(ctx).Err(err)
						return
					}
					response.Parse[core.module.RespRender](ctx, http.StatusOK, resp)
				default:
					errors.BadRequest(ctx, "invalid notification type")
				
			}
		},
		"notification",
	)
} 

func BuildHandler(config *router.Config, route... routen.Route) (*router.Handler, error) {
		handler := router.NewHandler(config)

		group := router.NewGroup(config.Name, route...)
	
		err := handler.Build(router.Base{
			Version:     *router.NewVersion(config.Version),
			Groups:      []router.Group{group},
		})
		if err != nil {
			return nil, err
		}
		return handler, nil
}
