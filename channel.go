package gosock

import (
	"bytes"
	"context"
	"encoding/json"
	"log"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type chanMessageType int

const (
	sendType chanMessageType = iota
	broadcastType
	replyType
)

type channelMessage struct {
	conn    *Conn
	msg     []byte
	msgType chanMessageType
	r       *Response
}

type Channel struct {
	path   string
	params *Params

	connections *ConnectionMap
	router      *Router
	send        chan *channelMessage
}

func (c *Channel) Path() string {
	return c.path
}

func (c *Channel) Params() *Params {
	return c.params
}

func (c *Channel) Param(key string) (string, bool) {
	return c.params.Get(key)
}

func newChannel(path string, params *Params, router *Router) *Channel {
	return &Channel{
		path:        path,
		params:      params,
		connections: newConnectionMap(),
		router:      router,
		send:        make(chan *channelMessage, 1),
	}
}

func (c *Channel) Reply(ctx context.Context, event string, payload interface{}) {
	conn := GetConnection(ctx)

	if conn == nil {
		return
	}

	var buf bytes.Buffer
	w := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	response := &Response{
		Channel: c.path,
		Event:   event,
		Payload: payload,
	}

	if err := encoder.Encode(response); err != nil {
		return
	}

	if err := w.Flush(); err != nil {
		return
	}

	chanMessage := &channelMessage{
		conn:    conn,
		msgType: replyType,
		msg:     buf.Bytes(),
	}

	c.send <- chanMessage
}

func (c *Channel) ReplyErr(ctx context.Context, err error) {
	c.Reply(ctx, "error", M{
		"error": err.Error(),
	})
}

func (c *Channel) Broadcast(ctx context.Context, event string, payload interface{}) {
	conn := GetConnection(ctx)

	if conn == nil {
		return
	}

	var buf bytes.Buffer
	w := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	response := &Response{
		Channel: c.path,
		Event:   event,
		Payload: payload,
	}

	if err := encoder.Encode(response); err != nil {
		return
	}

	if err := w.Flush(); err != nil {
		return
	}

	chanMessage := &channelMessage{
		conn:    conn,
		msgType: broadcastType,
		msg:     buf.Bytes(),
		r:       response,
	}

	c.send <- chanMessage
}

func (c *Channel) Emit(event string, payload interface{}) {
	var buf bytes.Buffer
	w := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	response := &Response{
		Channel: c.path,
		Event:   event,
		Payload: payload,
	}

	if err := encoder.Encode(response); err != nil {
		return
	}

	if err := w.Flush(); err != nil {
		return
	}

	chanMessage := &channelMessage{
		msgType: sendType,
		msg:     buf.Bytes(),
		r:       response,
	}

	c.send <- chanMessage
}

func (c *Channel) writer() {
	for msg := range c.send {
		msgPayload := msg.msg
		msgConn := msg.conn

		if msg.msgType == replyType && msgConn != nil {
			c.router.hub.pool.Schedule(func() {
				msgConn.sendRaw(msgPayload)
			})
			continue
		}

		c.connections.Lock()
		connections := c.connections.connections
		c.connections.Unlock()

	connWalk:
		for conn, ok := range connections {
			if !ok {
				continue connWalk
			}

			if msg.msgType == broadcastType && conn == msgConn {
				continue connWalk
			}

			// For closure
			sendConn := conn
			c.router.hub.pool.Schedule(func() {
				sendConn.sendRaw(msgPayload)
			})
		}
	}
}

func (c *Channel) handleJoin(ctx context.Context, msg *Message) {
	conn := GetConnection(ctx)
	if conn == nil {
		log.Printf("No connection in ctx")
		return
	}

	joinHandler, hasJoin := c.router.routerHandlers[joinEventName]

	if !hasJoin {
		log.Printf("Channel %s has no join handler", c.router.path)
		return
	}

	beforeJoin, hasBeforeJoin := c.router.routerHandlers[beforeJoinEventName]

	ctx = withMessage(ctx, msg)

	if hasBeforeJoin {
		err := beforeJoin(ctx, c)

		if err != nil {
			c.ReplyErr(ctx, err)
			return
		}
	}

	c.addConnection(conn)

	err := joinHandler(ctx, c)

	if err != nil {
		c.ReplyErr(ctx, err)
	}
}

func (c *Channel) handleLeave(ctx context.Context, msg *Message) {
	conn := GetConnection(ctx)

	if conn == nil {
		log.Printf("No connection in context")
		return
	}

	leavehandler, hasLeave := c.router.routerHandlers[leaveEventName]

	if hasLeave {
		ctx := withMessage(conn.ctx, msg)
		leavehandler(ctx, c)
	}

	c.removeConnection(conn)
}

func (c *Channel) handleDisconnect(conn *Conn) {
	handler, ok := c.router.routerHandlers[disconnectEventName]

	if ok {
		handler(conn.ctx, c)
	}

	c.removeConnection(conn)
}

func (c *Channel) addConnection(conn *Conn) {
	c.connections.add(conn)
	conn.channels[c] = true
}

func (c *Channel) removeConnection(conn *Conn) {
	log.Printf("Removing conn %s from channel %s", conn.Id, c.path)
	c.connections.del(conn)
	conn.channels[c] = false
	delete(conn.channels, c)
}
