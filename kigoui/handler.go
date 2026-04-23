package kigoui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	errcore "github.com/AgentNemo00/kigo-core/errors"
	"github.com/AgentNemo00/kigo-core/inquiry"
	"github.com/AgentNemo00/kigo-core/notification"
	"github.com/AgentNemo00/kigo-core/order"
	"github.com/AgentNemo00/kigo-core/ui"
	"github.com/AgentNemo00/kigo-core/update"
	"github.com/AgentNemo00/kigo/kigoui/frame"
	"github.com/AgentNemo00/kigo/kigoui/pubsub"
	"github.com/AgentNemo00/sca-instruments/log"
	ps "github.com/AgentNemo00/sca-instruments/pubsub"
	"github.com/AgentNemo00/sca-instruments/security"
	ringbuffer "github.com/EBWi11/mmap_ringbuffer"
)

type Handler struct {
	communication 		*pubsub.Communication
	config 				*Config
	ipc 				*frame.IPC
}

func NewHandler(config *Config) (*Handler, error) {
	communication, err := pubsub.NewCommunication(config.PubSubURL)
	if err != nil {
		return nil, err
	}
	ipc, err := frame.NewIPC(config.IPCPath)
	if err != nil {
		return nil, err
	}
	return &Handler{
		communication: communication,
		config: config,
		ipc: ipc,
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
				payload, ok := data.Payload.(inquiry.InquiryRenderPayload)
				if !ok && data.From != "" {
					log.Ctx(ctx).Error("Received invalid payload for NotificationReady: %v", data.Payload)
					if data.From != "" {
						h.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
					}
					return
				}
				if !h.IsConfigConform(ctx, payload) {
					log.Ctx(ctx).Warn("Received configuration is not adaptable: %v", payload)
					if data.From != "" {
						h.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
					}
					return 
				}
				switch(payload.Channel) {
					// TODO pubsub
					case ui.IPC:
						name, err := security.UUID()
						if err != nil {
							h.Error(ctx, data.From, errcore.Internal)
							return 
						}

						// TODO: context parrent
						ctxTransmission, cancel := context.WithCancel(context.TODO())
						ipc, err := h.ipc.Open(ctxTransmission, name, payload.MaxFrameSize)
						if err != nil {
							h.Error(ctx, data.From, errcore.Channel)
							log.Ctx(ctx).Error("ipc could not be open")
							return 
						}
						dataChan := make(chan []byte)
						close := func ()  {
							ipc.Close()
							cancel()
						}
						err = h.communication.PubModule.Publish(ctx, data.From, order.Order{
							From: h.config.Name,
							To: data.From,
							Order: order.OrderRender,
							Payload: order.OrderRenderPayload{
								ChannelName: name,
							},
						})
						if err != nil {
							log.Ctx(ctx).Err(err)
							return 
						}
						go h.Draw(ctxTransmission, dataChan)
						go h.Transmission(ctxTransmission, dataChan, ipc, payload, close)
						



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

func (h *Handler) Transmission(ctx context.Context, dataChan chan []byte, frames *frame.Frame, payload inquiry.InquiryRenderPayload, close func()) {
	started := time.Now()
	endAt := started.Add(payload.Time)
	bufferEmptyTimeout := started
	estimatedWaitingTime := time.Duration(0)
	if payload.FPS != 0 {
		estimatedWaitingTime = time.Duration(int(time.Second.Seconds())/payload.FPS)
	}

	for {
		select {
			case <- ctx.Done():
				return
			default:
		}
		data, err := frames.Read()
		if err == nil {
			dataChan <- data
			bufferEmptyTimeout = time.Now()
			continue
		}
		if errors.Is(err, ringbuffer.ErrBufferEmpty) && bufferEmptyTimeout.Add(payload.Timeout).After(time.Now()) {
			// TODO: send error state
			close()
			return 
		}
		if errors.Is(err, ringbuffer.ErrClosed) {
			// successful
			close()
			return
		}
		if started != endAt && endAt.After(time.Now()) {
			// successful if all the frames were send 
			close()
			return
		}
		time.Sleep(estimatedWaitingTime)
	}
	
}

func (h *Handler) Draw(ctx context.Context, dataChan chan []byte) {
	for {
		select {
		case <- ctx.Done():
			return
		case data, ok := <- dataChan:
			if !ok {
				return 
			}
			// TODO: draw
			log.Ctx(ctx).Debug("%v", data)			 
		default:
		}
	}
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