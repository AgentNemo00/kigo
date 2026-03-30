package config

import "time"

type Module struct {
	Name string
	Path string
	RebootsAllowed int // amount of reboots allowed before shutdown is called

	Init bool // ready to use
	Times
	CallingInterval time.Duration // how often the module should be called, if zero deactivated
	
	AmountOfReboots int
}

type Times struct {
	TimeReady time.Time // time it took to be ready
	TimeLastUpdate time.Time // time it was last updated
}	
