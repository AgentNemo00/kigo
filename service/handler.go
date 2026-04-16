package service

import (
	"context"
	"sync"
	"time"

	errcore "github.com/AgentNemo00/kigo-core/errors"
	"github.com/AgentNemo00/kigo-core/notification"
	"github.com/AgentNemo00/kigo-core/order"
	"github.com/AgentNemo00/kigo-core/update"
	"github.com/AgentNemo00/kigo-core/information"
	"github.com/AgentNemo00/kigo/module"
	"github.com/AgentNemo00/sca-instruments/log"
	"github.com/AgentNemo00/sca-instruments/pubsub"
)

type Handler struct {
	communication 	*module.Communication
	commander 		*Commander
	mu 				sync.RWMutex

	modules 		[]*module.Module
	renderTo		string
}

func (h *Handler) Start(ctx context.Context, name string, renderTo string) error {
	h.commander = &Commander{
		hostname: name,
		communication: h.communication,
	}
	h.renderTo = renderTo
	subscription, err := h.communication.Sub.Subscribe(ctx, name, func(ctx context.Context, metadata pubsub.Metadata, data *notification.Notification, responder pubsub.Responder[order.Order]) {
		if metadata.Error != nil {
			log.Ctx(ctx).Error("received error in message: %v", metadata.Error)
			return
		}
		log.Ctx(ctx).Debug("Got message at %s from %s", metadata.Timestamp.Format("15:04:05"), data.From)
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

func (h *Handler) GetModule(id string) (*module.Module, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for index, module := range h.modules {
		if module.ID == id {
			return module, index+1
		}
	}
	return nil, -1
}

func (h *Handler) CreateModule(name string) (*module.Module, int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	newMod, err := module.NewModule(name)
	if err != nil {
		return nil, -1, err
	}
	h.modules = append(h.modules, newMod)
	return newMod, len(h.modules)-1, nil
}

func (h *Handler) MainServiceWorker(ctx context.Context, data notification.Notification) {

	switch data.Notification {
		case notification.NotificationReady:
			notificationPayload, ok := data.Payload.(notification.NotificationReadyPayload)
			if !ok && data.From != "" {
				log.Ctx(ctx).Error("Received invalid payload for NotificationReady: %v", data.Payload)
				if data.From != "" {
					h.commander.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
				}
				return
			}
			h.NotificationReady(ctx, data, notificationPayload)
		case notification.NotificationUpdate:
			if data.From == "" {
				log.Ctx(ctx).Error("Sender not defined: %v", data.Payload)
				if data.To != "" {
					h.commander.Error(ctx, data.From, errcore.KiGoIDInvalid)
				}
				return
			}
			notificationPayload, ok := data.Payload.(notification.NotificationUpdatePayload)
			if !ok && data.From != "" {
				log.Ctx(ctx).Error("Received invalid payload for NotificationReady: %v", data.Payload)
				if data.From != "" {
					h.commander.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
				}
				return
			}
			h.NotificationUpdate(ctx, data, notificationPayload)
		case notification.NotificationInformation:
			if data.From == "" {
				log.Ctx(ctx).Error("Sender not defined: %v", data.Payload)
				if data.To != "" {
					h.commander.Error(ctx, data.From, errcore.KiGoIDInvalid)
				}
				return
			}
			notificationPayload, ok := data.Payload.(notification.NotificationInformationPayload)
			if !ok && data.From != "" {
				log.Ctx(ctx).Error("Received invalid payload for NotificationReady: %v", data.Payload)
				if data.From != "" {
					h.commander.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
				}
				return
			}
			h.NotificationInformation(ctx, data, notificationPayload)

		case notification.NotificationRender:
			// TODO: unsupported
		default:
			if data.From != "" {
				h.commander.Error(ctx, data.From, errcore.NotificationTypeInvalid)
			}
	}
} 

func (h *Handler) NotificationReady(ctx context.Context, data notification.Notification, payload notification.NotificationReadyPayload) {
	if data.From != "" {
		// module already exists
		moduleObj, _ := h.GetModule(data.From)
		if moduleObj == nil {
			log.Ctx(ctx).Error("Module not found: %s", data.From)
			h.commander.Error(ctx, data.From, errcore.ModuleNotFound)
			return
		}
		moduleObj.Times.StartUpDuration = payload.Duration
		moduleObj.Times.Heartbeat = payload.Heartbeat
		moduleObj.Changes = payload.Changes
		moduleObj.Ready = true
		h.commander.StartUp(ctx, data.From, order.OrderStartUpPayload{
			ID: moduleObj.ID,
			NumberOfModules: len(h.modules),
			MessageTo: order.MessageTo{
				Notification: h.commander.hostname,
				Render: h.renderTo,
			},
		})
	}
	// From empty
	moduleObj, index, err := h.CreateModule(payload.Name)
	if err != nil {
		log.Ctx(ctx).Err(err)
		// not sending anything as no sender information
		return
	}
	moduleObj.Times.StartUpDuration = payload.Duration
	moduleObj.Times.Heartbeat = payload.Heartbeat
	moduleObj.Changes = payload.Changes
	moduleObj.Ready = true
	h.commander.StartUp(ctx, data.From, order.OrderStartUpPayload{
		ID: moduleObj.ID,
		NumberOfModules: index+1,
		MessageTo: order.MessageTo{
			Notification: h.commander.hostname,
			Render: h.renderTo,
		},
	})
	// TODO: create go routine for checking heartbeat

}


func (h *Handler) NotificationUpdate(ctx context.Context, data notification.Notification, payload notification.NotificationUpdatePayload) {
	switch payload.Type {
		case update.Config:
			modConfig, ok := payload.Payload.(update.Module)
			if !ok {
				log.Ctx(ctx).Error("Received invalid payload for update.Config: %v", payload.Payload)
				if data.From != "" {
					h.commander.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
				}
				return
			}
			moduleObj, _ := h.GetModule(data.From)
			if moduleObj == nil {
				log.Ctx(ctx).Error("Module not found: %s", data.From)
				h.commander.Error(ctx, data.From, errcore.ModuleNotFound)
				return
			}
			moduleObj.Ready = modConfig.Ready
			moduleObj.Changes = modConfig.Changes
			moduleObj.TimeLastUpdate = time.Now()
			moduleObj.Times.Heartbeat = modConfig.Heartbeat
	default:
		h.commander.Error(ctx, data.To, errcore.NotificationTypeInvalid)
	}	
}

func (h *Handler) NotificationInformation(ctx context.Context, data notification.Notification, payload notification.NotificationInformationPayload) {
		switch payload.Type {
			case information.Modules:
				moduleInfos := make([]information.ModuleInformation, 0, len(h.modules))
				for _, mod := range h.modules {
					moduleInfos = append(moduleInfos, information.ModuleInformation{
						ID: mod.ID,
						Name: mod.Name,
						Changes: mod.Changes,
						Ready: mod.Ready,
						Heartbeat: mod.Heartbeat,
						LastHeartbeat: mod.Times.TimeLastUpdate,
					})
				}
				h.commander.Information(ctx, data.From, order.OrderInformationPayload{
					Payload: information.ModulesPayload{
						Modules: moduleInfos,
					},
				})
			case information.Module:
				payloadModule, ok := payload.Payload.(string)
				if !ok {
					log.Ctx(ctx).Error("Received invalid payload for information.Module: %v", payload.Payload)
					h.commander.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
					return
				}
				moduleObj, _ := h.GetModule(payloadModule)
				if moduleObj == nil {
					log.Ctx(ctx).Error("Module not found: %s", payloadModule)
					h.commander.Error(ctx, data.From, errcore.ModuleNotFound)
					return
				}
				h.commander.Information(ctx, data.From, order.OrderInformationPayload{
					Payload: information.ModuleInformation{
						ID: moduleObj.ID,
						Name: moduleObj.Name,
						Changes: moduleObj.Changes,
						Ready: moduleObj.Ready,
						Heartbeat: moduleObj.Heartbeat,
						LastHeartbeat: moduleObj.Times.TimeLastUpdate,
					},
				})
	
	default:
		h.commander.Error(ctx, data.To, errcore.NotificationTypeInvalid)
	}
}

