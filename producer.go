package gosock

import "context"

type Producer interface {
	Subscribe()
	Publish(ctx context.Context, path *Response)
	Stop()
}

type ProducerManager interface {
	Create(channel *Channel) Producer
}
