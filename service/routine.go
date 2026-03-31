package service

import (
	"context"
	"time"
	"github.com/agentnemo00/kigo-core/order"
	"github.com/agentnemo00/kigo/config"
	"github.com/AgentNemo00/sca-instruments/log"

)

type Routine struct {
	Module *config.Module
	Commander *Commander
}

func (r *Routine) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if r.Module.CallingInterval == 0 {
				return
			}
			if time.Now().After(r.Module.TimeLastUpdate.Add(r.Module.CallingInterval)) {
				msg := order.Order{
					From: r.Commander.hostname,
					To: r.Module.Name,	
					Order: order.OrderUpdate,
					Payload: nil,
				}
				err := r.Commander.communication.Pub.Publish(ctx, r.Module.Name, msg)
				if err != nil {
					log.Ctx(ctx).Error("Error while publishing: %s", err.Error())
				}
			}
		}
	}
}