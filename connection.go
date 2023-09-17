package gosock

import "context"

type Connection interface {
	Context() context.Context
}
