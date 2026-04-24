package frame

import (
	"context"
	"fmt"

	"github.com/AgentNemo00/sca-instruments/log"
	"github.com/AgentNemo00/sca-instruments/pubsub"
	"github.com/AgentNemo00/sca-instruments/pubsub/nats"
)


type PubSub struct {
	sub pubsub.Subscriber[[]byte]
}

func NewPubSub(url string) (*PubSub, error) {
	sub, err := nats.SubscriberWithURL[[]byte](url)
	if err != nil {
		return nil, err
	}
	return &PubSub{
		sub: sub,
	}, nil
}

func (p *PubSub) Open(ctx context.Context, name string) (*Frame, error) {
	frameChan := make(chan []byte)
	frameErr := make(chan error)
	frameHandler := &Frame{
		read: func() ([]byte, error) {
			for {
				// blocking
				select{
				case <- ctx.Done():
					return nil, ctx.Err()
				case data, ok := <- frameChan:
					if !ok {
						return nil, fmt.Errorf("data chan closed")
					}
					return data, nil
				case data, ok := <- frameErr:
					if !ok {
						return nil, fmt.Errorf("err chan closed")
					}
					return nil, data
				}
			}
		},
	}
	subscription, err := p.sub.Subscribe(ctx, name, func(ctx context.Context, metadata pubsub.Metadata, data *[]byte) {
		if metadata.Error != nil {
			log.Ctx(ctx).Error("received error in message: %v", metadata.Error)
			frameErr <- metadata.Error
			return
		}
		frameChan <- *data
	} )
	if err != nil {
		return nil, err
	}
	frameHandler.close = func ()  {
		subscription.Unsubscribe(ctx)
	}
	return frameHandler, nil
}

