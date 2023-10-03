package gosock

import (
	"context"
)

type BaseProducerManager struct {
}

func (bpm *BaseProducerManager) Init(hub *Hub) {}
func (bpm *BaseProducerManager) Start()        {}

func (bpm *BaseProducerManager) Create(channel *Channel) Producer {
	return NewBaseProducer(channel)
}

type BaseProducer struct {
	send    chan *ChannelMessage
	channel *Channel
}

func NewBaseProducer(channel *Channel) *BaseProducer {
	return &BaseProducer{
		send:    make(chan *ChannelMessage),
		channel: channel,
	}
}

func (bp *BaseProducer) Subscribe() {}

func (bp *BaseProducer) Stop() {}

// This shouldnt worry about sending to each connection. The channel is responsible for that
func (bp *BaseProducer) Publish(ctx context.Context, msg *ChannelMessage) error {
	bp.channel.Write(msg)

	return nil
}
