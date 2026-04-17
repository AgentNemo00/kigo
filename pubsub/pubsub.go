package pubsub

import (
	"github.com/AgentNemo00/sca-instruments/pubsub"
	"github.com/AgentNemo00/sca-instruments/pubsub/nats"
	"github.com/AgentNemo00/kigo-core/order"
	"github.com/AgentNemo00/kigo-core/notification"
)

type Communication struct {
	Pub pubsub.Publisher[order.Order]
	Sub pubsub.Subscriber[notification.Notification, order.Order]
	Subscription pubsub.Subscription
}

func NewCommunication(url string) (*Communication, error) {
	pub, err := nats.PublisherWithURL[order.Order](url)
	if err != nil {
		return nil, err
	}
	sub, err := nats.SubscriberWithURL[notification.Notification, order.Order](url)
	if err != nil {
		return nil, err
	}
	return &Communication{
		Pub: pub,
		Sub: sub,
	}, nil
}
