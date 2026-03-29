package core

import (
	"time"
)

const (
	// Basic Notifications
	NotificationStartUp = "StartUp"
	NotificationShutdown = "Shutdown"
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
	NotificationsSend []string // notification publish
	CallingDuration time.Duration // Interval in which the module should be updated without beeing called directly
}

type RespReboot struct {
	// Duration needed to reboot, when should notification startup be called
	Duration time.Duration
}

type RespUpdate struct {
	// Duration needed to update, when should notification render be called
	Duration time.Duration
	NotificationsSend []map[string]any // notification to publish
}

type RespRender struct {
	// where to render
	PositionX int
	PositionY int
	// object to render
	Object []byte
}