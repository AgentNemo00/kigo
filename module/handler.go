package module

import (
	"context"
	"sync"
	"time"

	"encoding/json"

	errcore "github.com/AgentNemo00/kigo-core/errors"
	"github.com/AgentNemo00/kigo-core/information"
	"github.com/AgentNemo00/kigo-core/inquiry"
	"github.com/AgentNemo00/kigo-core/notification"
	"github.com/AgentNemo00/kigo-core/order"
	"github.com/AgentNemo00/kigo-core/update"
	"github.com/AgentNemo00/kigo/pubsub"
	"github.com/AgentNemo00/sca-instruments/log"
	ps "github.com/AgentNemo00/sca-instruments/pubsub"
	"github.com/AgentNemo00/sca-instruments/security"
)

type Handler struct {
	communication 		*pubsub.Communication
	commander 			*Commander
	mu 					sync.RWMutex
    ctx 				context.Context
	
	lastHeartbeatCheck 	time.Time
	modules 			[]*Module
	config 				*Config
	db 					*Database
}

func NewHandler(config *Config) (*Handler, error) {
	db, err := NewDatabase(config.Database)
	if err != nil {
		return nil, err
	}
	communication, err := pubsub.NewCommunication(config.PubSubURL)
	if err != nil {
		return nil, err
	}
	return &Handler{
		communication: communication,
		modules: make([]*Module, 0),
		commander: NewCommander(config.Name, communication),
		lastHeartbeatCheck: time.Now(),
		config: config,
		db: db,
	}, nil
}

func (h *Handler) Start(ctx context.Context) error {
	h.ctx = ctx
	h.lastHeartbeatCheck = time.Now()
	modules, err := h.db.FindModulesDB(ctx)
	if err != nil {
		return err
	}
	h.modules = modules
	h.heartbeatRoutine(ctx)
	subscription, err := h.communication.Sub.Subscribe(ctx, h.commander.Name(), func(ctx context.Context, metadata ps.Metadata, data *notification.Notification) {
		defer func ()  {
			h.CheckHeartbeats()
		}()
		if metadata.Error != nil {
			log.Ctx(ctx).Error("received error in message: %v", metadata.Error)
			return
		}
		if data.From == "" {
			log.Ctx(ctx).Error("no receiver name given in message: %v", data)
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

func (h *Handler) Stop() {
	err := h.db.SaveModulesDB(h.ctx, h.modules)
	if err != nil {
		log.Ctx(h.ctx).Err(err)
	}
	h.communication.Subscription.Unsubscribe(h.ctx)
}

func (h *Handler) GetModule(id string) (*Module, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for index, module := range h.modules {
		if module.UUID == id {
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
		if m.UUID == mod.UUID {
			h.modules = append(h.modules[:i], h.modules[i+1:]...)
			break
		}
	}
}

func (h *Handler) MainServiceWorker(ctx context.Context, data notification.Notification) {
	log.Ctx(ctx).Debug("Got message: %#v", data)
	switch data.Notification {
		case notification.NotificationReady:
			var payload notification.NotificationReadyPayload
			err := mapToStruct(data.Payload, &payload)
			if err != nil {
				log.Ctx(ctx).Err(err)
				h.commander.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
				return
			}
			uuid, err := security.UUID()
			if err != nil {
				log.Ctx(ctx).Err(err)
				h.commander.Error(ctx, data.From, errcore.Internal)
				return				
			}
			var moduleObj *Module
			moduleObj, _ = h.GetModule(data.From)
			if moduleObj == nil {
				// no modules already with these id
				moduleObj, _, err = h.CreateModule(uuid)
				if err != nil {
					log.Ctx(ctx).Err(err)
					h.commander.Error(ctx, data.From, errcore.Internal)
					return				
				}				
				defer func ()  {
					err := h.db.CreateModuleDB(ctx, moduleObj)
					if err != nil {
						log.Ctx(ctx).Err(err)
					}
				}()
			} else {
				// already found module, refresh data
				uuid = data.From
				defer func ()  {
					err := h.db.SaveModuleDB(ctx, moduleObj)
					if err != nil {
						log.Ctx(ctx).Err(err)
					}
				}()
			}

			moduleObj.StartUpDuration = payload.Duration
			moduleObj.Heartbeat = payload.Heartbeat
			moduleObj.TimeLastUpdate = time.Now()
			moduleObj.Changes = payload.Changes
			moduleObj.Name = payload.Name
			moduleObj.Ready = true
			h.commander.StartUp(ctx, data.From, order.OrderStartUpPayload{
				ID: uuid,
				NumberOfModules: len(h.modules),
				MessageTo: order.MessageTo{
					Notification: h.config.Name,
					Render: h.config.RenderTo,
				},
			})
			log.Ctx(ctx).Debug("Send message to %s", data.From)
		case notification.NotificationUpdate:
			var payload notification.NotificationUpdatePayload
			err := mapToStruct(data.Payload, &payload)
			if err != nil {
				log.Ctx(ctx).Err(err)
				h.commander.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
				return
			}
			h.NotificationUpdate(ctx, data, payload)
		case inquiry.InquiryInformation:
			var payload inquiry.InquiryInformationPayload
			err := mapToStruct(data.Payload, &payload)
			if err != nil {
				log.Ctx(ctx).Err(err)
				h.commander.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
				return
			}
			h.NotificationInformation(ctx, data, payload)
		default:
			if data.From != "" {
				h.commander.Error(ctx, data.From, errcore.NotificationTypeInvalid)
			}
	}
} 

func (h *Handler) heartbeatRoutine(ctx context.Context) {
	ticker := time.NewTicker(time.Millisecond*100)
	defer ticker.Stop()

	for {
		select {
		case <- ctx.Done():
			return
		case <-ticker.C:
			h.CheckHeartbeats()
		}
	}
}

// check if any module is not responsive and delete it, then send shutdown command to it
func (h *Handler) CheckHeartbeats() {
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
		err := h.db.DeleteModuleDB(h.ctx, mod)
		if err != nil {
			log.Ctx(h.ctx).Err(err)
		}
		h.commander.Shutdown(h.ctx, mod.UUID)
	}
	h.lastHeartbeatCheck = time.Now()
}

func (h *Handler) NotificationUpdate(ctx context.Context, data notification.Notification, payload notification.NotificationUpdatePayload) {
	switch payload.Type {
		case update.Config:
			var modConfig update.Module
			err := mapToStruct(payload.Payload, &modConfig)
			if err != nil {
				log.Ctx(ctx).Err(err)
				h.commander.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
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
			moduleObj.Heartbeat = modConfig.Heartbeat
			err = h.db.SaveModuleDB(ctx, moduleObj)
			if err != nil {
				log.Ctx(ctx).Err(err)
			}
		case update.Heartbeat:
			modName, ok := payload.Payload.(string)
			if !ok {
				log.Ctx(ctx).Error("Received invalid payload for update.Config: %v", payload.Payload)
				if data.From != "" {
					h.commander.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
				}
				return
			}
			moduleObj, _ := h.GetModule(modName)
			if moduleObj == nil {
				log.Ctx(ctx).Error("Module not found: %s", data.From)
				h.commander.Error(ctx, data.From, errcore.ModuleNotFound)
				return
			}
			moduleObj.TimeLastUpdate = time.Now()
			log.Ctx(ctx).Debug("heartbeat for %s was update by %s", modName, data.From)
	default:
		h.commander.Error(ctx, data.To, errcore.NotificationTypeInvalid)
	}	
}

func (h *Handler) NotificationInformation(ctx context.Context, data notification.Notification, payload inquiry.InquiryInformationPayload) {
		defer func ()  {
			modObj, _ := h.GetModule(data.From)
			if modObj != nil {
				modObj.TimeLastUpdate = time.Now()
			}
		}()
		switch payload.Type {
			case information.Modules:
				moduleInfos := make([]information.ModuleInformation, 0, len(h.modules))
				for _, mod := range h.modules {
					moduleInfos = append(moduleInfos, information.ModuleInformation{
						ID: mod.UUID,
						Name: mod.Name,
						Changes: mod.Changes,
						Ready: mod.Ready,
						Heartbeat: mod.Heartbeat,
						LastHeartbeat: mod.TimeLastUpdate,
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
						ID: moduleObj.UUID,
						Name: moduleObj.Name,
						Changes: moduleObj.Changes,
						Ready: moduleObj.Ready,
						Heartbeat: moduleObj.Heartbeat,
						LastHeartbeat: moduleObj.TimeLastUpdate,
					},
				})
	
	default:
		h.commander.Error(ctx, data.To, errcore.NotificationTypeInvalid)
	}
}

func mapToStruct(m any, out any) error {
    data, err := json.Marshal(m)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, out)
}

