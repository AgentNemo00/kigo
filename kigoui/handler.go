package kigoui

import (
	"context"
	"encoding/binary"
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
	"github.com/AgentNemo00/kigo/kigoui/paint"
	"github.com/AgentNemo00/kigo/kigoui/pubsub"
	"github.com/AgentNemo00/sca-instruments/log"
	ps "github.com/AgentNemo00/sca-instruments/pubsub"
	"github.com/AgentNemo00/sca-instruments/security"
	ringbuffer "github.com/EBWi11/mmap_ringbuffer"
)

type Handler struct {
	communication 		*pubsub.Communication
	config 				*Config
	channelIPC 			*frame.IPC
	channelPubSub		*frame.PubSub
}

func NewHandler(config *Config) (*Handler, error) {
	communication, err := pubsub.NewCommunication(config.PubSubURL)
	if err != nil {
		return nil, err
	}
	var channelIPC *frame.IPC
	if config.IPCPath != "" {
		channelIPC, err = frame.NewIPC(config.IPCPath)
		if err != nil {
			return nil, err
		}		
	}
	channelPubSub, err := frame.NewPubSub(config.PubSubURL)
	if err != nil {
		return nil, err
	}
	return &Handler{
		communication: communication,
		config: config,
		channelIPC: channelIPC,
		channelPubSub: channelPubSub,
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
				// TODO: context
				err := h.StartRenderHandshake(context.TODO(), data.From, payload)
				if err != nil {
					log.Ctx(ctx).Err(err)
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

func (h *Handler) StartRenderHandshake(ctx context.Context, from string, payload inquiry.InquiryRenderPayload) error {
	name, err := security.UUID()
	if err != nil {
		h.Error(ctx, from, errcore.Internal)
		return err
	}

	ctxTransmission, cancel := context.WithCancel(ctx)
	dataChan := make(chan []byte)
	packageChan := make(chan paint.Package)
	var channel *frame.Frame
	channelClose := func ()  {
		channel.Close()
		cancel()
	}
	switch(payload.Channel) {
		case ui.IPC:
			channel, err = h.channelIPC.Open(ctxTransmission, name, payload.MaxFrameSize)
			if err != nil {
				h.Error(ctx, from, errcore.Channel)
				log.Ctx(ctx).Error("ipc could not be open")
				return err
			}

		case ui.PubSub:
			channel, err = h.channelPubSub.Open(ctxTransmission, name)
			if err != nil {
				h.Error(ctx, from, errcore.Channel)
				log.Ctx(ctx).Error("ipc could not be open")
				return err
			}
		default:
			h.Error(ctx, from, errcore.Unsupported)
			return fmt.Errorf("not supported channel")
		}
		err = h.communication.PubModule.Publish(ctx, from, order.Order{
			From: h.config.Name,
			To: from,
			Order: order.OrderRender,
			Payload: order.OrderRenderPayload{
				ChannelName: name,
			},
		})
		if err != nil {
			log.Ctx(ctx).Err(err)
			return err
		}
		go h.Draw(ctxTransmission, packageChan)
		go h.Transform(ctxTransmission, dataChan, payload.Format, packageChan)
		go h.Transmission(ctxTransmission, dataChan, channel, payload, channelClose)
		return nil
}

func (h *Handler) Transmission(ctx context.Context, dataChan chan []byte, frames *frame.Frame, payload inquiry.InquiryRenderPayload, close func()) error {
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
				return ctx.Err()
			default:
		}
		data, err := frames.Read()
		if err == nil {
			dataChan <- data
			bufferEmptyTimeout = time.Now()
			continue
		}
		if errors.Is(err, ringbuffer.ErrBufferEmpty) && bufferEmptyTimeout.Add(payload.Timeout).After(time.Now()) {
			close()
			return fmt.Errorf("transmission frame timeout")
		}
		if errors.Is(err, ringbuffer.ErrClosed) {
			// successful
			close()
			return nil
		}
		if started != endAt && endAt.After(time.Now()) {
			// error timeout 
			close()
			return fmt.Errorf("transmission timeout")
		}
		time.Sleep(estimatedWaitingTime)
	}
	
}

func (h *Handler) Transform(ctx context.Context, dataChan chan []byte, format string, packageChan chan paint.Package) error {
	for {
		select {
		case <- ctx.Done():
			return ctx.Err()
		case dataPackage, ok := <- dataChan:
			if !ok {
				return fmt.Errorf("channel transform closed")
			}
			positionX := binary.BigEndian.Uint16(dataPackage[0:2])
			positionY := binary.BigEndian.Uint16(dataPackage[2:4])
			size := binary.BigEndian.Uint32(dataPackage[4:8])
			data := dataPackage[8:size+8]
			if format != ui.RAW {
				// TODO: transform
			}
			packageChan <- paint.Package{
				PositionX: int(positionX),
				PositionY: int(positionY),
				Data: data,
			}
			log.Ctx(ctx).Debug("%v", dataPackage)			 
		default:
			log.Ctx(ctx).Debug("transform loop")
		}
	}
}

func (h *Handler) Draw(ctx context.Context, packageChan chan paint.Package) error {
		for {
		select {
		case <- ctx.Done():
			return ctx.Err()
		case data, ok := <- packageChan:
			if !ok {
				return fmt.Errorf("channel draw closed")
			}
			// TODO: draw
			log.Ctx(ctx).Debug("%v", data)
		default:
			log.Ctx(ctx).Debug("draw loop")
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