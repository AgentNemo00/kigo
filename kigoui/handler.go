package kigoui

import (
	"context"
	"fmt"
	"slices"

	errcore "github.com/AgentNemo00/kigo-core/errors"
	"github.com/AgentNemo00/kigo-core/inquiry"
	"github.com/AgentNemo00/kigo-core/notification"
	"github.com/AgentNemo00/kigo-core/order"
	"github.com/AgentNemo00/kigo-core/ui"
	"github.com/AgentNemo00/kigo-core/update"
	"github.com/AgentNemo00/kigo/kigoui/pubsub"
	"github.com/AgentNemo00/sca-instruments/log"
	ps "github.com/AgentNemo00/sca-instruments/pubsub"
	"github.com/AgentNemo00/sca-instruments/security"
)

type Handler struct {
	communication 		*pubsub.Communication
	config *Config
}

func NewHandler(config *Config) (*Handler, error) {
	communication, err := pubsub.NewCommunication(config.PubSubURL)
	if err != nil {
		return nil, err
	}
	return &Handler{
		communication: communication,
		config: config,
	}, nil
}

func (h *Handler) Start(ctx context.Context) error {
	subscription, err := h.communication.Sub.Subscribe(ctx, h.config.Name, func(ctx context.Context, metadata ps.Metadata, data *notification.Notification, responder ps.Responder[order.Order]) {
		if metadata.Error != nil {
			log.Ctx(ctx).Error("received error in message: %v", metadata.Error)
			return
		}
		if data.From == "" {
			log.Ctx(ctx).Error("no information from which module is send")
			return
		}
		log.Ctx(ctx).Debug("Got message at %s from %s", metadata.Timestamp.Format("15:04:05"), data.From)
		switch((*data).Notification) {
			case inquiry.InquiryRender:
				// TODO handshake
				notificationPayload, ok := data.Payload.(inquiry.InquiryRenderPayload)
				if !ok && data.From != "" {
					log.Ctx(ctx).Error("Received invalid payload for NotificationReady: %v", data.Payload)
					if data.From != "" {
						h.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
					}
					return
				}
				if !h.IsConfigConform(ctx, notificationPayload) {
					log.Ctx(ctx).Warn("Received configuration is not adaptable: %v", notificationPayload)
					if data.From != "" {
						h.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
					}
					return 
				}
				switch(notificationPayload.Channel) {
				case ui.IPC:
					name, err := security.UUID()
					if err != nil {
						// TODO: internal
					}
					// create IPC channel with the size needed
				}
			default:
				h.Error(ctx, data.From, errcore.NotificationTypeInvalid)
		}
	})
	if err != nil {
		return err
	}
	h.communication.Subscription = subscription
	return nil
}

func (h *Handler) IsConfigConform(ctx context.Context, payload inquiry.InquiryRenderPayload) bool {
	if h.config.FPS < payload.FPS {
		return false
	} 
	if slices.Index(h.config.Channels, payload.Channel) == -1 {
		return false
	}
	if slices.Index(h.config.Formats, payload.Format) == -1 {
		return false
	}
	return true
}

func (h *Handler) Heartbeat(ctx context.Context, to string) {
	h.communication.PubKigo.Publish(ctx, h.config.KiGo, notification.Notification{
		From: h.config.Name,
		To: h.config.KiGo,
		Notification: notification.NotificationUpdate,
		Payload: notification.NotificationUpdatePayload{
			Type: update.Heartbeat,
			Payload: to,
		},
	})
}

func (h *Handler) Error(ctx context.Context, to string, errorCore int) {
	msg := order.Order{
		From: h.config.Name,
		To: to,	
		Order: order.OrderError,
		Payload: order.OrderErrorPayload{
			Message: fmt.Sprintf("%d", errorCore),
		},
	}
	err := h.communication.PubModule.Publish(ctx, to, msg)
	if err != nil {
		log.Ctx(ctx).Err(err)
	}
}

func (h *Handler) Stop(ctx context.Context) {
	h.communication.Subscription.Unsubscribe(ctx)
}