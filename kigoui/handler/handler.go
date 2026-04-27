package handler

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	errcore "github.com/AgentNemo00/kigo-core/errors"
	"github.com/AgentNemo00/kigo-core/information"
	"github.com/AgentNemo00/kigo-core/inquiry"
	"github.com/AgentNemo00/kigo-core/notification"
	"github.com/AgentNemo00/kigo-core/order"
	"github.com/AgentNemo00/kigo-core/ui"
	"github.com/AgentNemo00/kigo-core/update"
	"github.com/AgentNemo00/kigo-ui/frame"
	"github.com/AgentNemo00/kigo-ui/paint"
	"github.com/AgentNemo00/kigo-ui/pubsub"
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
	ctx 				context.Context
}

func NewHandler(config *Config) (*Handler, error) {
	communication, err := pubsub.NewCommunication(config.PubSubUrl)
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
	channelPubSub, err := frame.NewPubSub(config.PubSubUrl)
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
	h.ctx = ctx
	subscription, err := h.communication.Sub.Subscribe(ctx, h.config.Name, func(ctx context.Context, metadata ps.Metadata, data *notification.Notification) {
		if metadata.Error != nil {
			log.Ctx(ctx).Error("received error in message: %v", metadata.Error)
			return
		}
		if data.From == "" {
			log.Ctx(ctx).Error("no receiver name given in message: %v", data)
			return
		}
		// defer h.Heartbeat(h.ctx, data.From)
		log.Ctx(ctx).Debug("Got message %v from %s", data, data.From)
		switch(data.Notification) {
			case inquiry.InquiryRender:
				var payload inquiry.InquiryRenderPayload
				err := mapToStruct(data.Payload, &payload)
				if err != nil {
					log.Ctx(ctx).Err(err)
					h.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
					return
				}
				if !h.IsConfigConform(ctx, payload) {
					log.Ctx(ctx).Warn("Received configuration is not adaptable: %v", payload)
					h.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
					return 
				}
				h.Heartbeat(h.ctx, data.From)
				err = h.StartRenderHandshake(h.ctx, data.From, payload)
				if err != nil {
					log.Ctx(ctx).Err(err)
				}
			case inquiry.InquiryInformation:
				var payload inquiry.InquiryInformationPayload
				err := mapToStruct(data.Payload, &payload)
				if err != nil {
					log.Ctx(ctx).Err(err)
					h.Error(ctx, data.From, errcore.NotificationPayloadInvalid)
					return
				}
				h.Heartbeat(h.ctx, data.From)
				log.Ctx(ctx).Debug("inquiry payload: %v", payload)
				switch (payload.Type) {
					case information.UI:
						log.Ctx(ctx).Debug("inquiry ui information")
						err := h.communication.PubModule.Publish(ctx, data.From, order.Order{
							From: h.config.Name,
							To: data.From,
							Order: order.OrderInformation,
							Payload: information.UIPayload{
								Channels: h.config.Channels,
								Formats: h.config.Formats,
							},
						})
						if err != nil {
							log.Ctx(ctx).Err(err)
						}
						log.Ctx(ctx).Debug("send ui information")
					case information.Screen:
						log.Ctx(ctx).Debug("inquiry screen information")
						width, height := GetScreenDimensions()
						err := h.communication.PubModule.Publish(ctx, data.From, order.Order{
							From: h.config.Name,
							To: data.From,
							Order: order.OrderInformation,
							Payload: information.ScreenPayload{
								Width: width,
								Height: height,
								MaxFPS: h.config.FPS,
							},
						})
						if err != nil {
							log.Ctx(ctx).Err(err)
						}
						log.Ctx(ctx).Debug("send screen information")
					default:
						h.Error(ctx, data.From, errcore.NotificationTypeInvalid)
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
		log.Ctx(ctx).Info("channel closed")
		channel.Close()
		cancel()
	}
	frameSize := payload.MaxFrameSize
	if frameSize <= 0 {
		// no frame size set. maximal possible frame size set
		width, height := GetScreenDimensions()
		frameSize = width * height + 8 // add header variables
	}
	switch(payload.Channel) {
		case ui.IPC:
			channel, err = h.channelIPC.Open(ctxTransmission, name, frameSize)
			if err != nil {
				h.Error(ctx, from, errcore.Channel)
				log.Ctx(ctx).Error("ipc could not be open")
				return err
			}
			log.Ctx(ctx).Info("IPC channel open with name: %s", name)
		case ui.PubSub:
			channel, err = h.channelPubSub.Open(ctxTransmission, name)
			if err != nil {
				h.Error(ctx, from, errcore.Channel)
				log.Ctx(ctx).Error("ipc could not be open")
				return err
			}
			log.Ctx(ctx).Info("PubSub channel open with name: %s", name)
		default:
			h.Error(ctx, from, errcore.Unsupported)
			return fmt.Errorf("not supported channel")
		}
		width, height := GetScreenDimensions()
		err = h.communication.PubModule.Publish(ctx, from, order.Order{
			From: h.config.Name,
			To: from,
			Order: order.OrderRender,
			Payload: order.OrderRenderPayload{
				ScreenWidth: width,
				ScreenHeight: height,
				MaxFrameSize: frameSize,
				ChannelName: channel.Name(),
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
	started := false
	startAt := time.Now()
	endAt := startAt
	bufferEmptyTimeout := startAt
	estimatedWaitingTime := time.Duration(0)
	if payload.FPS != 0 {
		estimatedWaitingTime = time.Duration(time.Millisecond*time.Duration(1000/payload.FPS))
	}
	log.Ctx(ctx).Debug("estimated sleeping: %d", estimatedWaitingTime.Milliseconds())
	for {
		log.Ctx(ctx).Debug("transmission loop")
		select {
			case <- ctx.Done():
				return ctx.Err()
			default:
		}
		data, err := frames.Read()
		if err == nil {
			log.Ctx(ctx).Debug("read amount of data: %d", len(data))
			dataChan <- data
			bufferEmptyTimeout = time.Now()
			if !started {
				started = true
				endAt = startAt.Add(payload.Time)
			}
		} else {
			log.Ctx(ctx).Debug("error during transmission")
			if errors.Is(err, ringbuffer.ErrBufferEmpty) && bufferEmptyTimeout.Add(payload.Timeout).After(time.Now()) && started {
				log.Ctx(ctx).Err(ringbuffer.ErrBufferEmpty)
				close()
				return fmt.Errorf("transmission frame timeout")
			}
			if errors.Is(err, ringbuffer.ErrClosed) {
				// successful
				log.Ctx(ctx).Debug("successful closed by module")
				close()
				return nil
			}
		}
		if startAt != endAt && endAt.Before(time.Now()) && started {
			// error timeout 
			close()
			err := fmt.Errorf("transmission timeout")
			log.Ctx(ctx).Err(err)
			return err
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
			if size < 1 {
				log.Ctx(ctx).Warn("size is zero, no data will be read")
				continue
			}
			log.Ctx(ctx).Debug("size of data: %d", size)
			data := dataPackage[8:size+8]
			log.Ctx(ctx).Debug("data: %v", data)
			if format != ui.RAW {
				// TODO: transform
			}
			packageChan <- paint.Package{
				PositionX: int(positionX),
				PositionY: int(positionY),
				Data: data,
			}
			log.Ctx(ctx).Debug("received data package %v, %s", dataPackage, string(data))			 
		default:
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
			log.Ctx(ctx).Debug("draw %v, %s", data, string(data.Data))
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
	log.Ctx(ctx).Debug("send heartbeat for %s", to)
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

func mapToStruct(m any, out any) error {
    data, err := json.Marshal(m)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, out)
}
