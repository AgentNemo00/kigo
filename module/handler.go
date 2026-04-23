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
	"github.com/AgentNemo00/sca-instruments/database"
	"github.com/AgentNemo00/sca-instruments/database/lectors"
	"github.com/AgentNemo00/sca-instruments/log"
	ps "github.com/AgentNemo00/sca-instruments/pubsub"
	"gorm.io/gorm"
)

type Handler struct {
	communication 		*pubsub.Communication
	commander 			*Commander
	mu 					sync.RWMutex
    ctx 				context.Context
	
	lastHeartbeatCheck 	time.Time
	modules 			[]*Module
	config 				*Config
	db 					*database.Database
}

func NewHandler(config *Config) (*Handler, error) {
	db, err := database.WithLector(lectors.SqliteByPath(config.Database))
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
	err := h.FindModulesDB(ctx)
	if err != nil {
		return err
	}
	subscription, err := h.communication.Sub.Subscribe(ctx, h.commander.Name(), func(ctx context.Context, metadata ps.Metadata, data *notification.Notification, responder ps.Responder[order.Order]) {
		defer func ()  {
			h.CheckHeartbeats()
			h.SaveModulesDB()
		}()
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

func (h *Handler) FindModulesDB(ctx context.Context) error {
	modules, err := gorm.G[Module](h.db.DB).Find(ctx)
	if err != nil {
		return err
	}
	for _, mod := range modules {
		h.modules = append(h.modules, &mod)
	}
	return nil
}

func (h *Handler) SaveModulesDB() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, mod := range h.modules {
		if mod.ID == 0 {
			err := gorm.G[Module](h.db.DB).Create(h.ctx, mod)
			if err != nil {
				return err
			}
			continue
		}
		_, err := gorm.G[Module](h.db.DB).Updates(h.ctx, *mod)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) DeleteModuleDB(ctx context.Context, mod *Module) error {
	if mod.ID <= 0 {
		return nil
	}
	_, err := gorm.G[Module](h.db.DB).Where("id = ?", mod.ID).Delete(ctx)
	return err
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
func (h *Handler) CheckHeartbeats() {
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
		h.DeleteModuleDB(h.ctx, mod)
		h.commander.Shutdown(h.ctx, mod.UUID)
	}
	h.lastHeartbeatCheck = time.Now()
}

// Set module ready and update its information
func (h *Handler) moduleSetReady (ctx context.Context, moduleObj *Module, startUpDuration time.Duration, heartbeat time.Duration, changes []string, length int) {
	moduleObj.Times.StartUpDuration = startUpDuration
	moduleObj.Times.Heartbeat = heartbeat
	moduleObj.Times.TimeLastUpdate = time.Now()
	moduleObj.Changes = changes
	moduleObj.Ready = true
	h.commander.StartUp(ctx, moduleObj.UUID, order.OrderStartUpPayload{
	ID: moduleObj.UUID,
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
			moduleObj.Times.TimeLastUpdate = time.Now()
			log.Ctx(ctx).Debug("heartbeat for %s was update by %s", modName, data.From)
	default:
		h.commander.Error(ctx, data.To, errcore.NotificationTypeInvalid)
	}	
}

func (h *Handler) NotificationInformation(ctx context.Context, data notification.Notification, payload inquiry.InquiryInformationPayload) {
		defer func ()  {
			modObj, _ := h.GetModule(data.From)
			if modObj != nil {
				modObj.Times.TimeLastUpdate = time.Now()
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
						ID: moduleObj.UUID,
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

