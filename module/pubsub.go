package module

import (
	"github.com/AgentNemo00/sca-instruments/pubsub"
	"github.com/AgentNemo00/sca-instruments/pubsub/nats"
	core "github.com/agentnemo00/kigo-core"
)

type Communication struct {
	Pub pubsub.Publisher[core.Order]
	Sub pubsub.Subscriber[core.Notification, core.Order]
	Subscription pubsub.Subscription
}

func NewCommunication(url string) (*Communication, error) {
	pub, err := nats.PublisherWithURL[core.Order](url)
	if err != nil {
		return nil, err
	}
	sub, err := nats.SubscriberWithURL[core.Notification, core.Order](url)
	if err != nil {
		return nil, err
	}
	return &Communication{
		Pub: pub,
		Sub: sub,
	}, nil
}
