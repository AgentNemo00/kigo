package config

import "time"

type Module struct {
	ID   string
	Name string
	RebootsAllowed int // amount of reboots allowed before shutdown is called

	Init bool // ready to use
	Times
	CallingInterval time.Duration // how often the module should be called, if zero deactivated
	
	AmountOfReboots int
}
type Times struct {
	TimeReady time.Duration // Duration to be ready
	TimeLastUpdate time.Time // time it was last updated
}

// unused
func (m *Module) Shutdown() {
	m.Init = false
	m.AmountOfReboots = 0
}

// unused
func (m *Module) Reboot() {
	m.Init = false
	m.AmountOfReboots++
}
