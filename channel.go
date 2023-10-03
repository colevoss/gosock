package gosock

import (
	"context"
	"errors"
	"log"
	"sync"
)

// Used to compare if two connections are the same
type ConnComparator func(*Conn, *Conn) bool

func defaultConnComparator(connA *Conn, connB *Conn) bool {
	return connA == connB
}

type Channel struct {
	sync.RWMutex

	path   string
	params *Params

	hub *Hub

	router *Router
	send   chan *ChannelMessage

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
		send:   make(chan *ChannelMessage, 1),

		hub: router.hub,

		conns: make(map[*Conn]bool),

		compareConnections: defaultConnComparator,
	}

	channel.producer = channel.hub.producerManager.Create(channel)
	channel.producer.Subscribe()

	return channel
}

func (c *Channel) Emit(ctx context.Context, event string, payload interface{}) error {
	response := &Response{
		Channel: c.path,
		Event:   event,
		Payload: payload,
	}

	return c.producer.Publish(context.Background(), EmitChannelMsg(GetConnection(ctx), response))
}

func (c *Channel) Reply(ctx context.Context, event string, payload interface{}) error {
	conn := GetConnection(ctx)

	if conn == nil {
		return errors.New("No connection available to reply to")
	}

	response := &Response{
		Channel: c.path,
		Event:   event,
		Payload: payload,
	}

	// Do not need to publish out to producer since its only responding to this connection
	c.sendMsg(ReplyChannelMsg(conn, response))

	return nil

	// return c.producer.Publish(context.Background(), ReplyChannelMsg(conn, response))
}

func (c *Channel) ReplyErr(ctx context.Context, err error) {
	c.Reply(ctx, "error", J{
		"error": err.Error(),
	})
}

func (c *Channel) Broadcast(ctx context.Context, event string, payload interface{}) error {
	conn := GetConnection(ctx)

	// if conn == nil {
	// 	return
	// }

	response := &Response{
		Channel: c.path,
		Event:   event,
		Payload: payload,
	}

	return c.producer.Publish(context.Background(), BroadcastChannelMsg(conn, response))
}

// Sends response to all connections on channel without publishing to producer
func (c *Channel) SendResp(response *Response) {
	chanMessage := &ChannelMessage{
		Type:     emitType,
		Response: response,
	}

	c.sendMsg(chanMessage)
}

func (c *Channel) Write(msg *ChannelMessage) {
	c.sendMsg(msg)
}

func (c *Channel) sendMsg(msg *ChannelMessage) {
	c.wg.Add(1)
	c.send <- msg
}

func (c *Channel) close() {
	c.wg.Wait()
	close(c.send)

	c.producer.Stop()

	c.router.removeChannel(c.path)
}

/**
 * Ran in a go routine to control the flow of messages to channel
 * connections by looping the `send` channel and handling outgoing
 * messages one-by-one
 */
func (c *Channel) writer() {
	for msg := range c.send {
		c.wg.Done()
		// TODO: Handle error
		msgPayload, _ := msg.Response.Encode()
		msgConn := msg.conn

		if msg.Type == replyType && msgConn != nil {
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

			if msg.Type == broadcastType && msgConn != nil && c.compareConnections(conn, msgConn) {
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
