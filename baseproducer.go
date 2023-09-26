package gosock

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type BaseProducerManager struct {
}

func (bpm *BaseProducerManager) Init(hub *Hub) {}
func (bpm *BaseProducerManager) Start()        {}

func (bpm *BaseProducerManager) Create(channel *Channel) Producer {
	return NewBaseProducer(channel)
}

type BaseProducer struct {
	send    chan *channelMessage
	channel *Channel
}

func NewBaseProducer(channel *Channel) *BaseProducer {
	return &BaseProducer{
		send:    make(chan *channelMessage),
		channel: channel,
	}
}

func (bp *BaseProducer) Subscribe() {}

func (bp *BaseProducer) Stop() {}

// This shouldnt worry about sending to each connection. The channel is responsible for that
func (bp *BaseProducer) Publish(ctx context.Context, resp *Response) {
	var buf bytes.Buffer
	w := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	if err := encoder.Encode(resp); err != nil {
		return
	}

	if err := w.Flush(); err != nil {
		return
	}

	bp.channel.Write(buf.Bytes())
}
