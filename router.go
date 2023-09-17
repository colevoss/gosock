package gosock

import (
	"context"
	"sync"
)

type EventHandler func(context.Context, *Channel) error

type RouterOnInit func(*Router)

type Router struct {
	sync.RWMutex
	hub      *Hub
	path     string
	channels map[string]*Channel
	handlers map[string]EventHandler

	routerHandlers map[string]EventHandler
}

func NewRouter(path string, hub *Hub) *Router {
	return &Router{
		path:           path,
		channels:       make(map[string]*Channel),
		handlers:       make(map[string]EventHandler),
		routerHandlers: make(map[string]EventHandler),
		hub:            hub,
	}
}

func (r *Router) On(handlerInit ...RouterOnInit) *Router {
	for _, handler := range handlerInit {
		handler(r)
	}

	return r
}

func (r *Router) Join(handler EventHandler) RouterOnInit {
	return func(router *Router) {
		// router.handlers[joinEventName] = handler
		router.routerHandlers[joinEventName] = handler
	}
}

func (r *Router) BeforeJoin(handler EventHandler) RouterOnInit {
	return func(router *Router) {
		// router.handlers[beforeJoinEventName] = handler
		router.routerHandlers[beforeJoinEventName] = handler
	}
}

func (r *Router) AfterLeave(handler EventHandler) RouterOnInit {
	return func(router *Router) {
		// router.handlers[afterLeaveEventName] = handler
		router.routerHandlers[afterLeaveEventName] = handler
	}
}

func (r *Router) Leave(handler EventHandler) RouterOnInit {
	return func(router *Router) {
		// router.handlers[leaveEventName] = handler
		router.routerHandlers[leaveEventName] = handler
	}
}

func (r *Router) Disconnect(handler EventHandler) RouterOnInit {
	return func(router *Router) {
		router.handlers[disconnectEventName] = handler
		router.routerHandlers[disconnectEventName] = handler
	}
}

func (r *Router) Event(event string, handler EventHandler) {
	r.handlers[event] = handler
}

func (r *Router) addChannel(path string, params *Params) *Channel {
	r.Lock()
	defer r.Unlock()

	channel := newChannel(path, params, r)
	r.channels[path] = channel

	go channel.writer()

	return channel
}
