package gosock

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gobwas/ws"
)

const connectEventName = "__connect__"

type ConnectionHandler func(conn *Conn)
type ServerEventInit func(hub *Hub)
type Middleware func(http.HandlerFunc) http.HandlerFunc

type Hub struct {
	sync.RWMutex

	pool *Pool

	channels *Node // value is meta channels
	conns    map[*Conn]bool

	connect    chan *Conn
	disconnect chan *Conn

	handlers    map[string]ConnectionHandler
	middlewares []Middleware

	handle http.HandlerFunc

	channelCache map[string]*Channel

	producerManager ProducerManager
}

func NewHub(pool *Pool) *Hub {
	hub := &Hub{
		channels:    NewTree(),
		conns:       make(map[*Conn]bool),
		connect:     make(chan *Conn),
		disconnect:  make(chan *Conn),
		handlers:    make(map[string]ConnectionHandler),
		pool:        pool,
		middlewares: []Middleware{},

		channelCache: make(map[string]*Channel),
	}

	hub.AddProducerManager(&BaseProducerManager{})

	return hub
}

func (h *Hub) AddProducerManager(manager ProducerManager) {
	h.Lock()
	defer h.Unlock()

	h.producerManager = manager
}

func (h *Hub) Use(middlewares ...Middleware) {
	h.middlewares = append(h.middlewares, middlewares...)
}

func Connect(handler ConnectionHandler) ServerEventInit {
	return func(h *Hub) {
		h.handlers[connectEventName] = handler
	}
}

func (h *Hub) Start() {
	go h.run()

	h.handle = h.wrapHandler()
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.handle == nil {
		panic("server handle is nil. Make sure server.start() is called")
	}

	h.handle.ServeHTTP(w, r)
}

func (h *Hub) wrapHandler() http.HandlerFunc {
	wrapped := http.HandlerFunc(h.handler)

	for i := len(h.middlewares) - 1; i >= 0; i-- {
		wrapped = h.middlewares[i](wrapped)
	}

	return wrapped
}

func (h *Hub) On(s ...ServerEventInit) *Hub {
	for _, initer := range s {
		initer(h)
	}

	return h
}

func (h *Hub) Channel(path string, handler func(*Router)) {
	router := NewRouter(path, h)
	handler(router)
	h.channels.Add(path, router)
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.connect:
			h.addConn(c)
			h.handleConnect(c)

		case c := <-h.disconnect:
			h.removeConn(c)
		}
	}
}

func (h *Hub) handleConnect(conn *Conn) {
	handler, ok := h.handlers[connectEventName]

	if !ok {
		return
	}

	go handler(conn)
}

func (h *Hub) Send(ctx context.Context, path string, event string, payload interface{}) {
	channel, ok := h.channelCache[path]

	if !ok {
		node, _ := h.channels.Lookup(path)

		if node == nil || node.Channel == nil {
			log.Printf("Channel not found %s", path)
			return
		}

		router := node.Channel
		channel, ok = router.channels[path]

		if !ok {
			log.Printf("Channel does not exist in router %s", path)
			return
		}
	}

	channel.Emit(ctx, event, payload)
}

func (h *Hub) handleMessage(conn *Conn, msg *Message) {
	channel, ok := h.channelCache[msg.Channel]

	if !ok {
		log.Printf("Channel not cached %s", msg.Channel)
		node, params := h.channels.Lookup(msg.Channel)

		if node == nil || node.Channel == nil {
			log.Printf("Channel not found %s", msg.Channel)
			return
		}

		router := node.Channel
		channel, ok = router.channels[msg.Channel]

		if !ok {
			channel = router.addChannel(msg.Channel, params)
			// This needs to be cleared at some point
			h.Lock()
			h.channelCache[msg.Channel] = channel
			h.Unlock()
		}
	}

	ctx := withConnection(conn.ctx, conn)

	switch msg.Event {
	case joinEventName:
		channel.handleJoin(ctx, msg)

	case leaveEventName:
		channel.handleLeave(ctx, msg)

	default:
		handler, ok := channel.router.handlers[msg.Event]

		if !ok {
			log.Printf("Channel does not have handler for event %s", msg.Event)
			return
		}

		// if hasConn := channel.connections.has(conn); !hasConn {
		if hasConn := channel.hasConn(conn); !hasConn {
			log.Printf("Connection %s does not exist in channel %s", conn.Id, msg.Channel)
			return
		}

		ctx := withMessage(ctx, msg)
		handler(ctx, channel)
	}
}

func (h *Hub) removeCachedChannel(path string) {
	h.Lock()
	defer h.Unlock()
	log.Printf("Removing cached channel %s", path)

	delete(h.channelCache, path)
}

func (h *Hub) handler(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)

	if err != nil {
		log.Fatalf("Error upgrading http request %v", err)
		return
	}

	ctx := r.Context()
	c := newConn(ctx, conn, h)

	h.connect <- c

	go c.read()
}

func (h *Hub) addConn(conn *Conn) {
	h.Lock()
	defer h.Unlock()

	h.conns[conn] = true
}

func (h *Hub) removeConn(conn *Conn) {
	h.Lock()
	defer h.Unlock()
	log.Printf("Removing connection")

	delete(h.conns, conn)
}

func newParam() interface{} {
	params := &Params{}

	return params
}
