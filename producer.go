package gosock

import "context"

type Producer interface {
	Subscribe()
	Publish(ctx context.Context, msg *ChannelMessage) error
	Stop()
}

type ProducerManager interface {
	Create(channel *Channel) Producer
}
