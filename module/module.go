package module

import (
	"time"
	"database/sql/driver"
	"github.com/AgentNemo00/sca-instruments/security"
	"gorm.io/gorm"
	"strings"
)

type Module struct {
	gorm.Model
	UUID   				string
	Name 				string
	Changes 			StringArray

	CreateAt 			time.Time // time it was created
	StartUpDuration 	time.Duration // Duration to be ready
	Heartbeat 			time.Duration // Duration between heartbeats
	TimeLastUpdate 		time.Time // time it was last updated	
	
	Ready 				bool

	SheduledForCleanUp 	bool
}

type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
    return strings.Join(s, ","), nil
}

func (s *StringArray) Scan(value interface{}) error {
    *s = strings.Split(value.(string), ",")
    return nil
}

func NewModule() (*Module, error) {
	uuid, err := security.UUID()
	if err != nil {
		return nil, err
	}
	return &Module{
		UUID:   uuid,
		CreateAt: time.Now(),
		TimeLastUpdate: time.Now(),
	}, nil
}

func (m *Module) LifecycleOver() bool {
	return m.TimeLastUpdate.Add(m.Heartbeat).Before(time.Now())
}
