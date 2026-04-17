package service

import (
	"context"
	"github.com/AgentNemo00/kigo/config"
	"fmt"
	"github.com/AgentNemo00/sca-instruments/api/router"
	"github.com/AgentNemo00/sca-instruments/log"
	"github.com/AgentNemo00/kigo/module"
	"github.com/AgentNemo00/kigo/pubsub"
)

type Service struct {
	config 			*config.Config
	handlerRest 	*router.Handler
	handler 		*module.Handler
	cancel 			context.CancelFunc
}

func NewService(config *config.Config) (*Service, error) {
	communication, err := pubsub.NewCommunication(config.PubSubUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create communication: %w", err)
	}
	return &Service{
		config: config,
		handler: module.NewHandler(communication),
	}, nil
}

func (s *Service) Start(ctxN context.Context) error {
	ctx, cancel := context.WithCancel(ctxN)
	s.cancel = cancel		
	err := s.handler.Start(ctx, s.config.Name, s.config.KiGoUIID)	
	if err != nil {
		return err
	}
	go func() {
		err := s.REST()	
		if err != nil {
			log.Ctx(ctx).Err(err)
		}
	}()
	<-ctx.Done()
	return ctx.Err()
}

func (s *Service) Stop(ctx context.Context) error {
	if s.handlerRest != nil {
		err := s.handlerRest.Stop(ctx)
		if err != nil {
			log.Ctx(ctx).Err(err)
		}
	}
	s.cancel()
	return nil
}

// just in case we have some external service pushing via rest
func (s *Service) REST() error {
	if !s.config.Ping && !s.config.Metrics && !s.config.Probes {
		return nil
	}
	s.handlerRest = router.NewHandler(&s.config.Config)
	err := s.handlerRest.Build(router.Simple())
	if err != nil {
		return err
	}
	err = s.handlerRest.Start()
	if err != nil {
		return err
	}
	return nil
}
