package core

import (
	"time"
)

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
	Payload map[string]any
}

// Responses wanted on Nofification send

// called after configutation readed
type RespStartUp struct {
	NotificationsOn []string // subcribe to
	NotificationsSend []string // notification send
	CallingDuration time.Duration // duration of the call, when should notification render be called
}

type RespReboot struct {
	// Duration needed to reboot, when should notification startup be called
	Duation time.Duration
}

type RespUpdate struct {
	// Duration needed to update, when should notification render be called
	Duration time.Duration
	NotificationsSend []string // notification send
}

type RespRender struct {
	// where to render
	PositionX int
	PositionY int
	// object to render
	// TODO: change
	Object any
}