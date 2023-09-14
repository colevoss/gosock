package gosock

import (
	"bytes"
	"encoding/json"
	"log"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type Channel struct {
	Path        string
	Params      *Params
	Connections *ConnectionMap
	router      *Router
}

func NewChannel(path string, params *Params, router *Router) *Channel {
	return &Channel{
		Path:        path,
		Params:      params,
		Connections: NewConnectionMap(),
		router:      router,
	}
}

func (c *Channel) Reply(conn *Conn, event string, payload interface{}) {
	response := &Response{
		Channel: c.Path,
		Event:   event,
		Payload: payload,
	}

	conn.Send(response)
}

func (c *Channel) ReplyErr(conn *Conn, err error) {
	resp := &Response{
		Channel: c.Path,
		Event:   "error",
		Payload: M{
			"error": err.Error(),
		},
	}

	conn.Send(resp)
}

func (c *Channel) BroadcastOld(conn *Conn, event string, payload interface{}) {
	response := &Response{
		Channel: c.Path,
		Event:   event,
		Payload: payload,
	}

	for connection, ok := range c.Connections.connections {
		if ok && connection != conn {
			connection.Send(response)
		}
	}
}

func (c *Channel) Broadcast(conn *Conn, event string, payload interface{}) {
	var buf bytes.Buffer
	w := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	response := &Response{
		Channel: c.Path,
		Event:   event,
		Payload: payload,
	}

	if err := encoder.Encode(response); err != nil {
		return
	}

	if err := w.Flush(); err != nil {
		return
	}

	for connection, ok := range c.Connections.connections {
		if ok && connection != conn {
			connection.sendRaw <- buf.Bytes()
		}
	}
}

func (c *Channel) Send(event string, payload interface{}) {
	response := &Response{
		Channel: c.Path,
		Event:   event,
		Payload: payload,
	}

	for connection, ok := range c.Connections.connections {
		if ok {
			connection.Send(response)
		}
	}
}

func (c *Channel) handleJoin(conn *Conn, msg *Message) {
	joinHandler, hasJoin := c.router.Handlers[Join]

	if !hasJoin {
		log.Printf("Channel %s has no join handler", c.router.Path)
		return
	}

	beforeJoin, hasBeforeJoin := c.router.Handlers[BeforeJoin]

	if hasBeforeJoin {
		err := beforeJoin(c, conn, msg)

		if err != nil {
			c.ReplyErr(conn, err)
			return
		}
	}

	c.addConnection(conn)

	err := joinHandler(c, conn, msg)

	if err != nil {
		c.ReplyErr(conn, err)
	}
}

func (c *Channel) handleLeave(conn *Conn, msg *Message) {
	leavehandler, hasLeave := c.router.Handlers[Leave]

	if hasLeave {
		leavehandler(c, conn, msg)
	}

	c.removeConnection(conn)
}

func (c *Channel) handleDisconnect(conn *Conn) {
	handler, ok := c.router.Handlers[Disconnect]

	if ok {
		handler(c, conn, nil)
	}

	c.removeConnection(conn)
}

func (c *Channel) addConnection(conn *Conn) {
	c.Connections.Add(conn)
	conn.channels[c] = true
}

func (c *Channel) removeConnection(conn *Conn) {
	log.Printf("Removing conn %s from channel %s", conn.Id, c.Path)
	c.Connections.Del(conn)
	conn.channels[c] = false
	delete(conn.channels, c)
}
