package gosock

import (
	"bytes"
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
	Path        string
	Params      *Params
	Connections *ConnectionMap
	router      *Router
	send        chan *channelMessage
}

func NewChannel(path string, params *Params, router *Router) *Channel {
	return &Channel{
		Path:        path,
		Params:      params,
		Connections: NewConnectionMap(),
		router:      router,
		send:        make(chan *channelMessage, 1),
	}
}

func (c *Channel) Reply(conn *Conn, event string, payload interface{}) {
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

	chanMessage := &channelMessage{
		conn:    conn,
		msgType: replyType,
		msg:     buf.Bytes(),
	}

	c.send <- chanMessage
}

func (c *Channel) ReplyErr(conn *Conn, err error) {
	// resp := &Response{
	// 	Channel: c.Path,
	// 	Event:   "error",
	// 	Payload: M{
	// 		"error": err.Error(),
	// 	},
	// }

	c.Reply(conn, "error", M{
		"error": err.Error(),
	})

	// chanMessage := &channelMessage{
	//   conn: conn,
	//   msgType: replyType,
	//   msg: buf.Bytes(),
	// }

	// conn.Send(resp)
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

	chanMessage := &channelMessage{
		conn:    conn,
		msgType: broadcastType,
		msg:     buf.Bytes(),
		r:       response,
	}

	c.send <- chanMessage
}

func (c *Channel) Send(event string, payload interface{}) {
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

		if msg.msgType == replyType {
			c.router.hub.pool.Schedule(func() {
				log.Printf("Sending msg to %s", msgConn.Id)
				msgConn.SendRaw(msgPayload)
			})
			// msgConn.SendRaw(msgPayload)
			continue
		}

		c.Connections.Lock()
		connections := c.Connections.connections
		c.Connections.Unlock()

	connWalk:
		for conn, ok := range connections {
			if !ok {
				continue connWalk
			}

			if msg.msgType == broadcastType && conn == msgConn {
				log.Printf("Not broadcasting to %s", conn.Id)
				continue connWalk
			}

			// For closure
			sendConn := conn
			c.router.hub.pool.Schedule(func() {
				log.Printf("!!!!!Sending msg to %s", sendConn.Id)
				sendConn.SendRaw(msgPayload)
			})
		}
	}
}

func (c *Channel) handleJoin(conn *Conn, msg *Message) {
	joinHandler, hasJoin := c.router.Handlers[JoinEventName]

	if !hasJoin {
		log.Printf("Channel %s has no join handler", c.router.Path)
		return
	}

	beforeJoin, hasBeforeJoin := c.router.Handlers[BeforeJoinEventName]

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
	leavehandler, hasLeave := c.router.Handlers[LeaveEventName]

	if hasLeave {
		leavehandler(c, conn, msg)
	}

	c.removeConnection(conn)
}

func (c *Channel) handleDisconnect(conn *Conn) {
	handler, ok := c.router.Handlers[DisconnectEventName]

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
