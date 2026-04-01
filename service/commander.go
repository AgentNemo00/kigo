package service

import (
	"context"
	"github.com/AgentNemo00/kigo/config"
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

func (c *Commander) Reboot(ctx context.Context, to string, moduleObj *config.Module) error {
	if moduleObj.AmountOfReboots >= moduleObj.RebootsAllowed {
		return c.Shutdown(ctx, to)
	}
	msg := order.Order{
		From: c.hostname,
		To: to,	
		Order: order.OrderReboot,
		Payload: nil,
	}
	return c.communication.Pub.Publish(ctx, to, msg)
}

func (c *Commander) Update(ctx context.Context, to string, payload any) error {
	msg := order.Order{
		From: c.hostname,
		To: to,	
		Order: order.OrderUpdate,
		Payload: payload,
	}
	return c.communication.Pub.Publish(ctx, to, msg)
}
