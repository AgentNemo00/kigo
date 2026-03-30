package config

import "time"

type Module struct {
	Name string
	Path string
	
	Times
	CallingInterval time.Duration
}

type Times struct {
	TimeReady time.Time
}	
