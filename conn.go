package gosock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

var connId = 0

type Conn struct {
	Id string
	sync.RWMutex

	conn net.Conn
	hub  *Hub

	channels map[*Channel]bool

	ctx context.Context
}

func (c *Conn) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}

	return context.Background()
}

func newConn(ctx context.Context, conn net.Conn, hub *Hub) *Conn {
	connection := &Conn{
		ctx:      ctx,
		Id:       fmt.Sprintf("conn-%d", connId),
		conn:     conn,
		hub:      hub,
		channels: make(map[*Channel]bool),
	}

	connId++

	return connection
}

func (c *Conn) close() {
	log.Printf("Closing connection %s", c.Id)
	c.conn.Close()

	for ch := range c.channels {
		c.hub.pool.Schedule(func() {
			ch.handleDisconnect(c)
		})
	}

	c.hub.disconnect <- c
}

func (c *Conn) WithContext(ctx context.Context) *Conn {
	if ctx == nil {
		panic("nil context")
	}

	c.ctx = ctx
	return c
}

func (c *Conn) read() {
	defer c.close()

	reader := wsutil.NewReader(c.conn, ws.StateServerSide)

	for {
		hdr, err := reader.NextFrame()

		if err != nil {
			log.Printf("Error reading client data %v", err)
			return
		}

		if hdr.OpCode == ws.OpClose {
			log.Printf("Closing code received %s", c.Id)
			return
		}

		if hdr.OpCode == ws.OpPing {
			log.Printf("PING")
		}

		var req Message

		decoder := json.NewDecoder(reader)
		if err := decoder.Decode(&req); err != nil {
			log.Printf("Error decoding client data %v", err)
			return
		}

		c.hub.pool.Schedule(func() {
			c.hub.handleMessage(c, &req)
		})
	}
}

func (c *Conn) sendRaw(msg []byte) {
	c.Lock()
	defer c.Unlock()

	_, err := c.conn.Write(msg)

	if err != nil {
		log.Printf("Error sending raw message %s", err)
	}
}

type ConnectionMap struct {
	sync.RWMutex
	connections map[*Conn]bool

	register   chan *Conn
	unregister chan *Conn
}

func newConnectionMap() *ConnectionMap {
	return &ConnectionMap{
		connections: make(map[*Conn]bool),
	}
}

func (cm *ConnectionMap) add(conn *Conn) {
	cm.Lock()
	defer cm.Unlock()

	cm.connections[conn] = true
}

func (cm *ConnectionMap) del(conn *Conn) {
	cm.Lock()
	defer cm.Unlock()

	delete(cm.connections, conn)
}

func (cm *ConnectionMap) has(conn *Conn) bool {
	cm.Lock()
	defer cm.Unlock()

	connVal, ok := cm.connections[conn]

	return ok && connVal
}
