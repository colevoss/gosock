package gosock

import (
	"sync"
)

type EventHandler func(*Channel, *Conn, *Message) error
type RouterOnInit func(*Router)

// type RouterRegister interface {
// 	Register(*Router)
// }

type Router struct {
	sync.RWMutex
	hub      *Hub
	Path     string
	Channels map[string]*Channel
	Handlers map[string]EventHandler
}

func NewRouter(path string, hub *Hub) *Router {
	return &Router{
		Path:     path,
		Channels: make(map[string]*Channel),
		Handlers: make(map[string]EventHandler),
		hub:      hub,
	}
}

func (r *Router) On(handlerInit ...RouterOnInit) *Router {
	for _, handler := range handlerInit {
		handler(r)
	}

	return r
}

func Join(handler EventHandler) RouterOnInit {
	return func(router *Router) {
		router.Handlers[JoinEventName] = handler
	}
}

func BeforeJoin(handler EventHandler) RouterOnInit {
	return func(router *Router) {
		router.Handlers[BeforeJoinEventName] = handler
	}
}

func AfterLeave(handler EventHandler) RouterOnInit {
	return func(router *Router) {
		router.Handlers[AfterLeaveEventName] = handler
	}
}

func Leave(handler EventHandler) RouterOnInit {
	return func(router *Router) {
		router.Handlers[LeaveEventName] = handler
	}
}

func Disconnect(handler EventHandler) RouterOnInit {
	return func(router *Router) {
		router.Handlers[DisconnectEventName] = handler
	}
}

func (r *Router) Event(event string, handler EventHandler) {
	r.Handlers[event] = handler
}

func (r *Router) addChannel(path string, params *Params) *Channel {
	// Maybe lock this??
	r.Lock()
	defer r.Unlock()

	channel := NewChannel(path, params, r)
	r.Channels[path] = channel

	go channel.writer()

	return channel
}
