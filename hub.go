package gosock

import (
	"log"
	"net/http"

	"github.com/gobwas/ws"
)

const ConnectEventName = "__connect__"

type ConnectionHandler func(conn *Conn)
type ServerEventInit func(hub *Hub)

type Hub struct {
	pool *Pool

	channels    *Node // value is meta channels
	connections *ConnectionMap

	connect    chan *Conn
	disconnect chan *Conn

	handlers map[string]ConnectionHandler
}

func NewHub(pool *Pool) *Hub {
	return &Hub{
		channels:    NewTree(),
		connections: newConnectionMap(),
		connect:     make(chan *Conn),
		disconnect:  make(chan *Conn),
		handlers:    make(map[string]ConnectionHandler),
		pool:        pool,
	}
}

func Connect(handler ConnectionHandler) ServerEventInit {
	return func(h *Hub) {
		h.handlers[ConnectEventName] = handler
	}
}

func (h *Hub) Start() {
	log.Printf("Starting Hub Server")

	go h.run()

	http.ListenAndServe(":8080", http.HandlerFunc(h.handler))
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
			h.connections.add(c)
			h.handleConnect(c)

		case c := <-h.disconnect:
			h.connections.del(c)
		}
	}
}

func (h *Hub) handleConnect(conn *Conn) {
	handler, ok := h.handlers[ConnectEventName]

	if !ok {
		return
	}

	go handler(conn)
}

func (h *Hub) handleMessage(conn *Conn, msg *Message) {
	node, params := h.channels.Lookup(msg.Channel)

	if node == nil || node.Channel == nil {
		log.Printf("Channel not found %s", msg.Channel)
		return
	}

	metaChannel := node.Channel
	channel, ok := metaChannel.channels[msg.Channel]

	if !ok {
		channel = metaChannel.addChannel(msg.Channel, params)
	}

	ctx := withConnection(conn.ctx, conn)

	// switch msg.Event {
	switch msg.Event {
	case joinEventName:
		channel.handleJoin(ctx, msg)

	case leaveEventName:
		channel.handleLeave(ctx, msg)

	default:
		handler, ok := metaChannel.handlers[msg.Event]

		if !ok {
			log.Printf("Channel does not have handler for event %s", msg.Event)
			return
		}

		if hasConn := channel.connections.has(conn); !hasConn {
			log.Printf("Connection %s does not exist in channel %s", conn.Id, msg.Channel)
			return
		}

		ctx := withMessage(ctx, msg)
		handler(ctx, channel)
	}
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
