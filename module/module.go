package module

import (
	"time"

	"github.com/AgentNemo00/sca-instruments/security"
	"gorm.io/gorm"
)

type Module struct {
	gorm.Model
	UUID   string
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
	uuid, err := security.UUID()
	if err != nil {
		return nil, err
	}
	return &Module{
		UUID:   uuid,
		Name: name,
		Times: Times{
			CreateAt: time.Now(),
			TimeLastUpdate: time.Now(),
		},
	}, nil
}

func (m *Module) LifecycleOver() bool {
	return m.TimeLastUpdate.Add(m.Heartbeat).Before(time.Now())
}
