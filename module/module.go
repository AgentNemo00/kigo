package module

import (
	"time"

	"github.com/AgentNemo00/sca-instruments/security"
)

type Module struct {
	ID   string
	Name string
	Changes []string

	Times
	
	Ready bool

	SheduledForCleanUp bool
}

type Times struct {
	CreateAt time.Time // time it was created
	StartUpDuration time.Duration // Duration to be ready
	Heartbeat time.Duration // Duration between heartbeats
	TimeLastUpdate time.Time // time it was last updated
}

func NewModule(name string) (*Module, error) {
	id, err := security.UUID()
	if err != nil {
		return nil, err
	}
	return &Module{
		ID:   id,
		Name: name,
		Times: Times{
			CreateAt: time.Now(),
			TimeLastUpdate: time.Now(),
		},
	}, nil
}