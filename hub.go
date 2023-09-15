package gosock

import (
	"log"
	"net/http"

	"github.com/gobwas/ws"
)

const Connect = "__connect__"

type ConnectionHandler func(conn *Conn)

type Hub struct {
	Channels    *Node // value is meta channels
	Connections *ConnectionMap

	connect    chan *Conn
	disconnect chan *Conn

	handlers map[string]ConnectionHandler
	pool     *Pool
}

func NewHub(pool *Pool) *Hub {
	return &Hub{
		Channels:    NewTree(),
		Connections: NewConnectionMap(),
		connect:     make(chan *Conn),
		disconnect:  make(chan *Conn),
		handlers:    make(map[string]ConnectionHandler),
		pool:        pool,
	}
}

func (h *Hub) Start() {
	log.Printf("Starting Hub Server")

	go h.Run()

	http.ListenAndServe(":8080", http.HandlerFunc(h.handler))
}

func (h *Hub) On(event string, handler ConnectionHandler) {
	h.handlers[event] = handler
}

func (h *Hub) Channel(path string, handler func(*Router)) {
	router := NewRouter(path, h)
	handler(router)
	h.Channels.Add(path, router)
}

func (h *Hub) Run() {
	log.Printf("Starting Hub")

	for {
		select {
		case c := <-h.connect:
			h.Connections.Add(c)
			h.handleConnect(c)

		case c := <-h.disconnect:
			h.Connections.Del(c)
		}
	}
}

func (h *Hub) handleConnect(conn *Conn) {
	handler, ok := h.handlers[Connect]

	if !ok {
		return
	}

	go handler(conn)
}

func (h *Hub) handleMessage(conn *Conn, msg *Message) {
	h.Channels.Print()
	node, params := h.Channels.Lookup(msg.Channel)

	if node == nil || node.Channel == nil {
		log.Printf("Channel not found %s", msg.Channel)
		return
	}

	metaChannel := node.Channel
	channel, ok := metaChannel.Channels[msg.Channel]

	if !ok {
		channel = metaChannel.addChannel(msg.Channel, params)
	}

	switch msg.Event {
	case JoinEventName:
		channel.handleJoin(conn, msg)

	case LeaveEventName:
		channel.handleLeave(conn, msg)

	default:
		handler, ok := metaChannel.Handlers[msg.Event]

		if !ok {
			log.Printf("Channel does not have handler for event %s", msg.Event)
			return
		}

		if hasConn := channel.Connections.Has(conn); !hasConn {
			log.Printf("Connection %s does not exist in channel %s", conn.Id, msg.Channel)
			return
		}

		handler(channel, conn, msg)
	}
}

func (h *Hub) handler(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)

	if err != nil {
		log.Fatalf("Error upgrading http request %v", err)
		return
	}

	c := NewConn(conn, h)

	h.connect <- c

	go c.Read()
	// go c.Write()
}
