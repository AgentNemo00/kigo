package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AgentNemo00/kigo/config"
	"github.com/AgentNemo00/sca-instruments/errors"
	"github.com/AgentNemo00/sca-instruments/log"
	"github.com/AgentNemo00/sca-instruments/pubsub"
	"github.com/AgentNemo00/kigo-core/notification"
	"github.com/AgentNemo00/kigo-core/order"
	"github.com/AgentNemo00/kigo/module"
)

type Handler struct {
	communication 	*module.Communication
	modules 		[]config.Module
	commander 		*Commander
	mu 				sync.RWMutex
}

func (h *Handler) Start(ctx context.Context, name string) error {
	h.commander = &Commander{
		hostname: name,
		communication: h.communication,
	}
	subscription, err := h.communication.Sub.Subscribe(ctx, name, func(ctx context.Context, metadata pubsub.Metadata, data *notification.Notification, responder pubsub.Responder[order.Order]) {
		if metadata.Error != nil {
			log.Ctx(ctx).Error("received error in message: %v", metadata.Error)
			return
		}
		if data.From == "" {
			log.Ctx(ctx).Error("no sender data in message: %v", data)
			return
		}
		h.MainServiceWorker(ctx, *data)
	})
	if err != nil {
		return err
	}
	h.communication.Subscription = subscription
	return nil
}

func (h *Handler) Stop(ctx context.Context) {
	h.communication.Subscription.Unsubscribe(ctx)
}

func (h *Handler) GetModule(name string) (*config.Module, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for index, module := range h.modules {
		if module.Name == name {
			return &module, index+1
		}
	}
	return nil, -1
}

func (h *Handler) CreateModule(name string) (*config.Module, int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	module, index := h.GetModule(name)
	if module != nil {
		return module, index, errors.New(fmt.Errorf("a module with this name already exits"))
	}
	newModule := config.Module{
		Name: name,
	}
	h.modules = append(h.modules, newModule)
	return &newModule, len(h.modules), nil
}

func (h *Handler) MainServiceWorker(ctx context.Context, data notification.Notification) {
	if data.From == "" {
		log.Ctx(ctx).Error("Sender not defined: %v", data.Payload)
		return
	}
	switch data.Notification {
		case notification.NotificationReady:
			notificationPayload, ok := data.Payload.(notification.NotificationReadyPayload)
			if !ok {
				log.Ctx(ctx).Error("Received invalid payload for NotificationReady: %v", data.Payload)
				err := h.commander.Shutdown(ctx, data.From)
				if err != nil {
					log.Ctx(ctx).Err(err)
				}
				return
			}
			h.NotificationReady(ctx, data, notificationPayload)
		case notification.NotificationUpdate:
			moduleObj, _ := h.GetModule(data.From)
			if moduleObj == nil {
				log.Ctx(ctx).Info("Module not found: %s", data.From)
				err := h.commander.Shutdown(ctx, data.From)
				if err != nil {
					log.Ctx(ctx).Err(err)
				}
			}
			notificationPayload, ok := data.Payload.(notification.NotificationUpdatePayload)
			if !ok {
				log.Ctx(ctx).Error("Received invalid payload for NotificationUpdate: %v", data.Payload)
				err := h.commander.Reboot(ctx, data.From, moduleObj)
				if err != nil {
					log.Ctx(ctx).Err(err)
				}
				return
			}
			h.NotificationUpdate(ctx, data, notificationPayload)
		case notification.NotificationRender:
			moduleObj, _ := h.GetModule(data.From)
			if moduleObj == nil {
				log.Ctx(ctx).Info("Module not found: %s", data.From)
				err := h.commander.Shutdown(ctx, data.From)
				if err != nil {
					log.Ctx(ctx).Err(err)
				}
			}
			_, ok := data.Payload.(notification.NotificationRenderPayload)
			if !ok {
				log.Ctx(ctx).Error("Received invalid payload for NotificationRender: %v", data.Payload)
				err := h.commander.Reboot(ctx, data.From, moduleObj)
				if err != nil {
					log.Ctx(ctx).Err(err)
				}
				return
			}
			// TODO: Use a file to share UI element ref. Do some security checks
		default:
			log.Ctx(ctx).Error("Received unknown notification: %s", data.Notification)
	}
} 

func (h *Handler) NotificationReady(ctx context.Context, data notification.Notification, payload notification.NotificationReadyPayload) {
	var err error
	moduleObj, index := h.GetModule(data.From)
	if moduleObj == nil {
		log.Ctx(ctx).Info("Module not found: %s", data.From)
		// if no name giving in the config 
		moduleObj, index, err = h.CreateModule(data.From)
		if err != nil {
			log.Ctx(ctx).Error("Could not create new module: %s", data.From)
			err := h.commander.Shutdown(ctx, data.From)
			if err != nil {
				log.Ctx(ctx).Err(err)
			}
			return
		}		
	}
	moduleObj.TimeReady = payload.Duration
	orderPayload := order.OrderStartUpPayload{
		QueuePosition: index,
	}
	err = h.communication.Pub.Publish(ctx, data.From, order.Order{
		From: h.commander.hostname,
		To: data.From,
		Order: order.OrderStartUp,
		Payload: orderPayload,
	})
	if err != nil {
		log.Ctx(ctx).Err(err)
	}
	moduleObj.Init = true // initialized
	if moduleObj.CallingInterval != 0 {
		moduleObj.CallingInterval = payload.CallingInterval
		routine := Routine{Module: moduleObj, Commander: h.commander}
		routine.Start(ctx)
	}
}

func (h *Handler) NotificationUpdate(ctx context.Context, data notification.Notification, payload notification.NotificationUpdatePayload) {
	currentTime := time.Now()
	moduleObj, _ := h.GetModule(data.From)
	if moduleObj == nil {
		log.Ctx(ctx).Info("Module not found: %s", data.From)
		return
	}
	viewer := Viewer{SizeX: 1000, SizeY: 1000}
	SizeX, SizeY, err := viewer.GetDimensions(ctx) // TODO: communicate with the UI service
	if err != nil {
		log.Ctx(ctx).Error("Failed to get dimensions: %v", err)
		err := h.commander.Update(ctx, data.From, map[string]string{"Command": "retry"})
		if err != nil {
			log.Ctx(ctx).Err(err)
		}
		return
	}
	orderPayload := order.OrderRenderPayload{
		SizeX: SizeX,
		SizeY: SizeY,
	}
	durationenNeeded := time.Since(currentTime)
	defer func() {
    	moduleObj.TimeLastUpdate = time.Now()
	}()

	if payload.Duration == 0 {
		go func ()  {
			err := h.commander.Update(ctx, data.From, orderPayload)
			if err != nil {
				log.Ctx(ctx).Error("Error while publishing: %s", err.Error())
				return
			}					
		}()
		return
	}
	waitingDurationen := payload.Duration - durationenNeeded
	if waitingDurationen < 0 {
		waitingDurationen = 0
	}
	time.AfterFunc(waitingDurationen, func() {
		err := h.commander.Update(ctx, data.From, orderPayload)
		if err != nil {
			log.Ctx(ctx).Error("Error while publishing: %s", err.Error())
			return
		}
	})
}
