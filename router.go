package gosock

import (
	"log"
	"sync"
)

type EventHandler func(*Channel, *Conn, *Message) error

type RouterRegister interface {
	Register(*Router)
}

type Router struct {
	sync.RWMutex
	Path     string
	Channels map[string]*Channel
	Handlers map[string]EventHandler
}

func NewRouter(path string) *Router {
	return &Router{
		Path:     path,
		Channels: make(map[string]*Channel),
		Handlers: make(map[string]EventHandler),
	}
}

func (r *Router) Event(event string, handler EventHandler) {
	log.Printf("Adding event %s to channel %s", event, r.Path)
	r.Handlers[event] = handler
}

func (r *Router) addChannel(path string, params *Params) *Channel {
	// Maybe lock this??
	log.Printf("Creating channel %s for meta %s", path, r.Path)
	r.Lock()
	defer r.Unlock()

	channel := NewChannel(path, params, r)
	r.Channels[path] = channel
	return channel
}
