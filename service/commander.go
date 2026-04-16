package service

import (
	"context"
	"fmt"

	"github.com/AgentNemo00/kigo-core/order"
	"github.com/AgentNemo00/kigo/module"
	"github.com/AgentNemo00/sca-instruments/log"
)

type Commander struct {
	hostname 		string
	communication 	*module.Communication
}

func (c *Commander) Shutdown(ctx context.Context, to string) {
	msg := order.Order{
		From: c.hostname,
		To: to,
		Order: order.OrderShutdown,
		Payload: nil,
	}
	err := c.communication.Pub.Publish(ctx, to, msg)
	if err != nil {
		log.Ctx(ctx).Err(err)
	}
}

func (c *Commander) StartUp(ctx context.Context, to string, payload order.OrderStartUpPayload) {
	msg := order.Order{
		From: c.hostname,
		To: to,
		Order: order.OrderStartUp,
		Payload: payload,
	}
	err := c.communication.Pub.Publish(ctx, to, msg)
		if err != nil {
		log.Ctx(ctx).Err(err)
	}
}


func (c *Commander) Information(ctx context.Context, to string, payload any) {
	msg := order.Order{
		From: c.hostname,
		To: to,	
		Order: order.OrderInformation,
		Payload: payload,
	}
	err := c.communication.Pub.Publish(ctx, to, msg)
	if err != nil {
		log.Ctx(ctx).Err(err)
	}

}

func (c *Commander) Error(ctx context.Context, to string, errorCore int) {
	msg := order.Order{
		From: c.hostname,
		To: to,	
		Order: order.OrderError,
		Payload: order.OrderErrorPayload{
			Message: fmt.Sprintf("%d", errorCore),
		},
	}
	err := c.communication.Pub.Publish(ctx, to, msg)
	if err != nil {
		log.Ctx(ctx).Err(err)
	}
}