package service

import (
	"context"
	"github.com/AgentNemo00/sca-instruments/config"
	"fmt"
	"github.com/AgentNemo00/sca-instruments/nats"
	"github.com/AgentNemo00/sca-instruments/api/router"
	"github.com/agentnemo00/kigo/module"
)

type Service struct {
	config 			*config.Config
	handlerRest 	*router.Handler
	handler 		*Handler
	cancel 			context.CancelFunc
}

func NewService(config *config.Config) (*Service, error) {
	communication, err := module.NewCommunication(config.Config.PubSubUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create communication: %w", err)
	}
	return &Service{
		config: config,
		handler: &Handler{
			communication: communication,
			modules: config.Modules,
		},
	}, nil
}

func (s *Service) Start(ctxN context.Context) error {
	ctx, cancel := context.WithCancel(ctxN)
	s.cancel = cancel		
	err := s.handler.Start(ctx, s.config.Name)	
	if err != nil {
		return err
	}
	go s.REST()
	<-ctx.Done()
	return ctx.Err()
}

func (s *Service) Stop(ctx context.Context) error {
	if s.handlerRest != nil {
		s.handlerRest.Stop(ctx)
	}
	s.cancel()
	return nil
}

// just in case we have some external service pushing via rest
func (s *Service) REST() error {
	if s.config.Ping == false && s.config.Metrics == false && s.config.Probes == false {
		return nil
	}
	s.handlerRest = router.NewHandler(s.config.Config)
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
