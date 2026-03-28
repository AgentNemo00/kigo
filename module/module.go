package module

import (
	"time"
)

type ModuleConfig struct {
	Name string
	Port int
}

func (mc *ModuleConfig) Default() {
	if mc.Port == 0 {
		mc.Port = 10001
	}
	if mc.Name == "" {
		panic("Module name cannot be empty")
	}
}

type Config struct {
	Modules []ModuleConfig
}

func (c *Config) Default() {
	if len(c.Modules) == 0 {
		c.Modules = []ModuleConfig{
			{
				Name: "KigoTextModule",
				Port: 10001,
			},
		}
		// TODO: trigger Kigo Text
	}
}

const (
	// ModuleName is the name of the module, used for identification and logging.
	NotificationStartUp = "StartUp"
	NotifcationShutdown = "Shutdown"
	NotificationReboot  = "Reboot"
	NotificationUpdate  = "Update"
	NotificationRender  = "Render"
)

// Payloads for notifications

type PayloadStartUp struct {
	// Position of the module
	QueuePosition int
}

type PayloadRender struct {
	SizeX int
	SizeY int
}

type PayloadUpdate struct {
	Payload any
}

// Responses wanted on Nofification send

// called after configutation readed
type RespStartUp struct {
	NotificationsOn []string // subcribe to
}

type RespReboot struct {
	// Duration needed to reboot, when should notification startup be called
	Duation time.Duration
}

type RespUpdate struct {
	// Duration needed to update, when should notification render be called
	Duration time.Duration
}

type RespRender struct {
	// where to render
	PositionX int
	PositionY int
	// object to render
	Object any
}



