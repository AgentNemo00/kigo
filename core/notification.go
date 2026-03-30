package core

// Messages send from the module

import (
	"time"
)	

const (
    // Basic notifications from the modules to the main service
	NotificationReady 	= "NotificationReady" // called when the module is ready to receive orders, i.e. after startup procedore, Publish to main
	NotificationUpdate 	= "NotificationUpdate" // called when the module has an update, i.e. update data or state, Publisch to main
	NotificationRender 	= "NotificationRender" // called when the module needs to render something, i.e. after update or order, Publish to main
)

type Notification struct {
	From 			string
	To 				string
	Notification	string
	Payload 		any
}

type NotificationReadyPayload struct {
	Duration time.Duration // Duration needed to be ready, when should order startup be called
	CallingInterval time.Duration // Interval in which the module should be updated without beeing called directly
}

type NotificationUpdatePayload struct {
	Duration time.Duration // Duration needed to update, when should notification render be called
}

type NotificationRenderPayload struct {
	// where to render
	PositionX int
	PositionY int

	Object []byte // object to render
	Path string // path to the object to render, if object is empty, path should be used
}
