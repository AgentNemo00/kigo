package service

import (
	"context"
	"github.com/AgentNemo00/sca-instruments/config"
	"time"
	"fmt"
	"github.com/AgentNemo00/sca-instruments/nats"
	"github.com/agentnemo00/kigo-core/notification"
	"github.com/agentnemo00/kigo-core/order"
	"github.com/AgentNemo00/sca-instruments/pubsub"
	"github.com/AgentNemo00/sca-instruments/log"
	"github.com/AgentNemo00/sca-instruments/api/router"
	"github.com/agentnemo00/kigo/module"
)

type Service struct {
	config 			*config.Config
	communication 	*module.Communication
	cancel 			context.CancelFunc
	handler 		*router.Handler
}

func NewService(config *config.Config) (*Service, error) {
	communication, err := module.NewCommunication(config.Config.PubSubUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create communication: %w", err)
	}
	return &Service{
		config: config,
		communication: communication,
	}, nil
}

func (s *Service) Start(ctxN context.Context) error {
	ctx, cancel := context.WithCancel(ctxN)
	s.cancel = cancel		
	subscription, err := s.communication.Sub.Subscribe(ctx, s.config.Name, func(ctx context.Context, metadata pubsub.Metadata, data *notification.Notification, responder pubsub.Responder[order.Order]) {
		if metadata.Error != nil {
			log.Ctx(ctx).Errorf("Received error in message: %v", metadata.Error)
			return
		}
		s.MainServiceWorker(ctx, *msg)
	})
	if err != nil {
		return err
	}
	s.communication.Subscription = subscription
	go s.REST()
	<-ctx.Done()
}

func (s *Service) Stop(ctx context.Context) error {
	if s.handler != nil {
		s.handler.Stop(ctx)
	}
	s.communication.Subscription.Unsubscribe(ctx)
	s.cancel()
	return nil
}

// just in case we have some external service pushing via rest
func (s *Service) REST()  {
	if s.config.Ping == false && s.config.Metrics == false && s.config.Probes == false {
		return
	}
	handler := router.NewHandler(s.config.Config)
	err := apiHandler.Build(router.Simple())
	if err != nil {
		log.Fatal(err)
		return
	}
	err = apiHandler.Start()
	if err != nil {
		log.Fatal(err)
	}
}

func (s *Service) MainServiceWorker(ctx context.Context, data notification.Notification) { {
	moduleObject, index := s.config.GetModule(data.From)
	if moduleObject == nil && data.Notification != notification.NotificationReady {
		log.Ctx(ctx).Errorf("Received notification from unknown module: %s", notificationPayload.From)
		s.ShutdownOrder(ctx, notificationPayload.From)
		return
	}
	switch data.Notification {
		case notification.NotificationReady:
			notificationPayload, ok := data.Payload.(notification.NotificationReadyPayload)
			if !ok {
				log.Ctx(ctx).Errorf("Received invalid payload for NotificationReady: %v", data.Payload)
				s.ShutdownOrder(ctx, notificationPayload.From)
				return
			}
			if moduleObject != nil {
				// if no name giving
				moduleObject, err := s.config.CreateModule(data.From)
				if err != nil {
					log.Ctx(ctx).Errorf("Could not create new module: %s", data.From)
					s.ShutdownOrder(ctx, notificationPayload.From)
					return
				}					
			}
			moduleObject.TimeReady = notificationPayload.Duration
			moduleObject.CallingInterval = notificationPayload.CallingInterval
			orderPayload := order.OrderStartUpPayload{
				QueuePosition: index,
			}
			err := s.communication.Pub.Publish(ctx, data.From, order.Order{
				From: s.config.Name,
				To: data.From,
				Order: order.OrderStartUp,
				Payload: orderPayload,
			})
			if err != nil {
				log.Ctx(ctx).Error("Error while publishing: %s", err.Error())
				return
			}
			moduleObject.Init = true
			moduleObject.AmountOfReboots = 0
			if moduleObject.CallingInterval != 0 {
				s.TakeInitiative(ctx, moduleObject)
			}
		case notification.NotificationUpdate:
			notificationPayload, ok := data.Payload.(notification.NotificationUpdatePayload)
			if !ok {
				log.Ctx(ctx).Errorf("Received invalid payload for NotificationUpdate: %v", data.Payload)
				s.RebootOrder(ctx, data.From, moduleObject)
				return
			}
			SizeX, SizeY, err := s.GetDimensions(ctx)
			if err != nil {
				log.Ctx(ctx).Errorf("Failed to get dimensions: %v", err)
				s.UpdateOrder(ctx, data.From)
				return
			}
			orderPayload := order.OrderRenderPayload{
				SizeX: SizeX,
				SizeY: SizeY,
			}
			if notificationPayload.Duration == 0 {
				go func ()  {
					err := s.PublishOrder(ctx, data.From, orderPayload)
					if err != nil {
						log.Ctx(ctx).Error("Error while publishing: %s", err.Error())
						return
					}					
				}()
			} else {
				time.AfterFunc(notificationPayload.Duration, func() {
					err := s.PublishOrder(ctx, data.From, orderPayload)
					if err != nil {
						log.Ctx(ctx).Error("Error while publishing: %s", err.Error())
						return
					}
				})
			}
			moduleObject.TimeLastUpdate = time.Now()
		case notification.NotificationRender:
			notificationPayload, ok := data.Payload.(notification.NotificationRenderPayload)
			if !ok {
				log.Ctx(ctx).Errorf("Received invalid payload for NotificationRender: %v", data.Payload)
				s.RebootOrder(ctx, data.From, moduleObject)
				return
			}
			// TODO: forward to UI service create a repo for kigo-ui
		default:
			log.Ctx(ctx).Errorf("Received unknown notification: %s", data.Notification)

	}
}

func (s *Service) TakeInitiative(ctx context.Context, moduleObject config.Module) {
	for {
		select {
		case <-ctx.Done():
			return
		case moduleObject.CallingInterval == 0:
			return
		default:
			if time.Now().After(moduleObject.TimeLastUpdate.Add(moduleObject.CallingInterval)) {
				msg := order.Order{
					From: s.config.Name,
					To: moduleObject.Name,	
					Order: order.UpdateOrder,
					Payload: nil,
				}
				err := s.communication.Pub.Publish(ctx, to, msg)
				if err != nil {
					log.Ctx(ctx).Error("Error while publishing: %s", err.Error())
				}
			}
		}
	}
}

func (s *Service) UpdateOrder(ctx context.Context, to string) error {
	msg := order.Order{
		From: s.config.Name,
		To: to,	
		Order: order.UpdateOrder,
		Payload: nil,
	}
	return s.communication.Pub.Publish(ctx, to, msg)
}

func (s *Service) RebootOrder(ctx context.Context, to string, moduleObject config.Module) error {
	if moduleObject.AmountOfReboots >= moduleObject.RebootsAllowed {
		return s.ShutdownOrder(ctx, to)
	}
	msg := order.Order{
		From: s.config.Name,
		To: to,	
		Order: order.OrderReboot,
		Payload: nil,
	}
	return s.communication.Pub.Publish(ctx, to, msg)
}

func (s *Service) ShutdownOrder(ctx context.Context, to string) error {
	msg := order.Order{
		From: s.config.Name,
		To: to,
		Order: order.OrderShutdown,
		Payload: nil,
	}
	return s.communication.Pub.Publish(ctx, to, msg)
}

func (s *Service) GetDimensions(ctx context.Context) (int, int, error) {
	// TODO: implement dimension retrieval from UI service
	return 1000, 1000, nil
}

func (s *Service) RenderOrder(ctx context.Context, to string, payload order.OrderRenderPayload) error {
	msg := order.Order{
		From: s.config.Name,
		To: to,
		Order: order.OrderRender,
		Payload: payload,
	}
	return s.communication.Pub.Publish(ctx, to, msg)
}
