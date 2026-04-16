package service

import (
	"context"
	"fmt"

	"github.com/AgentNemo00/kigo-core/order"
	"github.com/AgentNemo00/kigo/module"
)

type Commander struct {
	hostname 		string
	communication 	*module.Communication
}

func (c *Commander) Shutdown(ctx context.Context, to string) error {
	msg := order.Order{
		From: c.hostname,
		To: to,
		Order: order.OrderShutdown,
		Payload: nil,
	}
	return c.communication.Pub.Publish(ctx, to, msg)
}

func (c *Commander) StartUp(ctx context.Context, to string, payload order.OrderStartUpPayload) error {
	msg := order.Order{
		From: c.hostname,
		To: to,
		Order: order.OrderStartUp,
		Payload: payload,
	}
	return c.communication.Pub.Publish(ctx, to, msg)
}


func (c *Commander) Information(ctx context.Context, to string, payload any) error {
	msg := order.Order{
		From: c.hostname,
		To: to,	
		Order: order.OrderInformation,
		Payload: payload,
	}
	return c.communication.Pub.Publish(ctx, to, msg)
}

func (c *Commander) Error(ctx context.Context, to string, err int) error {
	msg := order.Order{
		From: c.hostname,
		To: to,	
		Order: order.OrderError,
		Payload: order.OrderErrorPayload{
			Message: fmt.Sprintf("%d", err),
		},
	}
	return c.communication.Pub.Publish(ctx, to, msg)
}