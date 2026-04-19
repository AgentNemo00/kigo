package module

import (
	"context"
	"sync"
	"time"

	errcore "github.com/AgentNemo00/kigo-core/errors"
	"github.com/AgentNemo00/kigo-core/information"
	"github.com/AgentNemo00/kigo-core/inquiry"
	"github.com/AgentNemo00/kigo-core/notification"
	"github.com/AgentNemo00/kigo-core/order"
	"github.com/AgentNemo00/kigo-core/update"
	"github.com/AgentNemo00/kigo/pubsub"
	"github.com/AgentNemo00/sca-instruments/log"
	ps "github.com/AgentNemo00/sca-instruments/pubsub"
)

type Handler struct {
	communication 		*pubsub.Communication
	commander 			*Commander
	mu 					sync.RWMutex
    ctx 				context.Context
	
	lastHeartbeatCheck 	time.Time
	modules 			[]*Module
	config 				*Config
}

func NewHandler(config *Config) (*Handler, error) {
	communication, err := pubsub.NewCommunication(config.PubSubURL)
	if err != nil {
		return nil, err
	}
	return &Handler{
		communication: communication,
		modules: make([]*Module, 0),
		commander: NewCommander(config.Name, communication),
		config: config,
	}, nil
}

func (h *Handler) Start(ctx context.Context) error {
	h.ctx = ctx
	h.lastHeartbeatCheck = time.Now()
	subscription, err := h.communication.Sub.Subscribe(ctx, h.commander.Name(), func(ctx context.Context, metadata ps.Metadata, data *notification.Notification, responder ps.Responder[order.Order]) {
		defer h.CheckHeartbeats(ctx)
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

func (h *Handler) GetModule(id string) (*Module, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for index, module := range h.modules {
		if module.ID == id {
			return module, index+1
		}
	}
	return nil, -1
}

func (h *Handler) CreateModule(name string) (*Module, int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	newMod, err := NewModule(name)
	if err != nil {
		return nil, -1, err
	}
	h.modules = append(h.modules, newMod)
	return newMod, len(h.modules)-1, nil
}

func (h *Handler) DeleteModule(mod *Module) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, m := range h.modules {
		if m.ID == mod.ID {
			h.modules = append(h.modules[:i], h.modules[i+1:]...)
			break
		}
	}
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
		case inquiry.InquiryInformation:
			if data.From == "" {
				log.Ctx(ctx).Error("Sender not defined: %v", data.Payload)
				if data.To != "" {
					h.commander.Error(ctx, data.From, errcore.KiGoIDInvalid)
				}
				return
			}
			notificationPayload, ok := data.Payload.(inquiry.InquiryInformationPayload)
			if !ok && data.From != "" {
				log.Ctx(ctx).Error("Received invalid payload for NotificationReady: %v", data.Payload)
				if data.From != "" {
					h.commander.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
				}
				return
			}
			h.NotificationInformation(ctx, data, notificationPayload)

		case inquiry.InquiryRender:
			// TODO: unsupported
			fallthrough
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
		h.moduleSetReady(ctx, moduleObj, payload.Duration, payload.Heartbeat, payload.Changes, len(h.modules))
	}
	// From empty
	moduleObj, index, err := h.CreateModule(payload.Name)
	if err != nil {
		log.Ctx(ctx).Err(err)
		// not sending anything as no sender information
		return
	}
	h.moduleSetReady(ctx, moduleObj, payload.Duration, payload.Heartbeat, payload.Changes, index+1)
}

// check if any module is not responsive and delete it, then send shutdown command to it
func (h *Handler) CheckHeartbeats(ctx context.Context) {
	// Do not need to check more often than every 16ms (60fps)
	if h.lastHeartbeatCheck.Add(time.Millisecond*16).Before(time.Now()) {
		return
	}
	toDelete := make([]*Module, 0)
	for _, mod := range h.modules {
		if !mod.LifecycleOver() {
			continue
		}
		toDelete = append(toDelete, mod)
	}
	for _, mod := range toDelete {
		log.Ctx(h.ctx).Info("Module %s is not responsive, deleting it", mod.Name)
		h.DeleteModule(mod)
		h.commander.Shutdown(ctx, mod.ID)
	}
	h.lastHeartbeatCheck = time.Now()
}

// Set module ready and update its information
func (h *Handler) moduleSetReady (ctx context.Context, moduleObj *Module, startUpDuration time.Duration, heartbeat time.Duration, changes []string, length int) {
	moduleObj.Times.StartUpDuration = startUpDuration
	moduleObj.Times.Heartbeat = heartbeat
	moduleObj.Changes = changes
	moduleObj.Ready = true
	h.commander.StartUp(ctx, moduleObj.ID, order.OrderStartUpPayload{
	ID: moduleObj.ID,
	NumberOfModules: length,
	MessageTo: order.MessageTo{
		Notification: h.commander.Name(),
		Render: h.config.RenderTo,
	},
	UIconfiguration: order.UIConfiguration{

	},
	})
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

func (h *Handler) NotificationInformation(ctx context.Context, data notification.Notification, payload inquiry.InquiryInformationPayload) {
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

