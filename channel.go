package gosock

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type chanMessageType int

const (
	sendType chanMessageType = iota
	broadcastType
	replyType
)

// Used to compare if two connections are the same
type ConnComparator func(*Conn, *Conn) bool

func defaultConnComparator(connA *Conn, connB *Conn) bool {
	return connA == connB
}

type channelMessage struct {
	conn    *Conn
	msg     []byte
	msgType chanMessageType
	r       *Response
}

type Channel struct {
	sync.RWMutex

	path   string
	params *Params

	hub *Hub

	router *Router
	send   chan *channelMessage

	conns map[*Conn]bool

	compareConnections ConnComparator

	producer Producer

	wg sync.WaitGroup
}

func (c *Channel) Path() string {
	return c.path
}

func (c *Channel) Params() *Params {
	return c.params
}

func (c *Channel) Param(key string) (string, bool) {
	if c.params == nil {
		return "", false
	}

	return c.params.Get(key)
}

func newChannel(path string, params *Params, router *Router) *Channel {
	channel := &Channel{
		path:   path,
		params: params,
		router: router,
		send:   make(chan *channelMessage, 1),

		hub: router.hub,

		conns: make(map[*Conn]bool),

		compareConnections: defaultConnComparator,
	}

	channel.producer = channel.hub.producerManager.Create(channel)
	channel.producer.Subscribe()

	return channel
}

func (c *Channel) TestProd(ctx context.Context, event string, payload interface{}) {
	response := &Response{
		Channel: c.path,
		Event:   event,
		Payload: payload,
	}

	c.producer.Publish(context.Background(), response)
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

	c.sendMsg(chanMessage)
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

	c.sendMsg(chanMessage)
	// c.send <- chanMessage
}

func (c *Channel) SendResp(response *Response) {
	var buf bytes.Buffer
	w := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

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

	c.sendMsg(chanMessage)
	// c.send <- chanMessage
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

	c.sendMsg(chanMessage)
	// c.send <- chanMessage
}

func (c *Channel) Write(data []byte) {
	c.sendMsg(
		&channelMessage{
			msgType: sendType,
			msg:     data,
		},
	)
}

func (c *Channel) sendMsg(msg *channelMessage) {
	c.wg.Add(1)
	c.send <- msg
}

func (c *Channel) close() {
	c.wg.Wait()
	log.Printf("Closing channel %s", c.path)
	close(c.send)
	c.producer.Stop()
	c.router.removeChannel(c.path)
}

func (c *Channel) writer() {
	for msg := range c.send {
		c.wg.Done()
		msgPayload := msg.msg
		msgConn := msg.conn

		if msg.msgType == replyType && msgConn != nil {
			c.hub.pool.Schedule(func() {
				msgConn.sendRaw(msgPayload)
			})
			continue
		}

		c.RLock()
		conns := c.conns
		c.RUnlock()

	connWalk:
		for conn, ok := range conns {
			if !ok {
				continue connWalk
			}

			if msg.msgType == broadcastType && c.compareConnections(conn, msgConn) {
				continue connWalk
			}

			// For closure
			sendConn := conn
			c.hub.pool.Schedule(func() {
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
	c.removeConnection(conn)

	handler, ok := c.router.routerHandlers[disconnectEventName]

	if ok {
		handler(conn.ctx, c)
	}
}

func (c *Channel) addConnection(conn *Conn) {
	c.Lock()
	defer c.Unlock()

	c.conns[conn] = true
	conn.addChannel(c)
}

func (c *Channel) removeConnection(conn *Conn) {
	c.Lock()
	defer c.Unlock()

	delete(c.conns, conn)
	conn.removeChannel(c)

	if (len(c.conns)) == 0 {
		go c.close()
	}

	// if len(c.conns) == 0 {
	// 	c.close()
	// }
}

func (c *Channel) hasConn(conn *Conn) bool {
	c.RLock()
	defer c.RUnlock()

	connBool, exists := c.conns[conn]

	if exists {
		return connBool
	}

	return false
}
