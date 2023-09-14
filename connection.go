package gosock

import (
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

	conn    net.Conn
	send    chan *Response
	sendRaw chan []byte
	hub     *Hub

	channels map[*Channel]bool
}

func NewConn(conn net.Conn, hub *Hub) *Conn {
	connection := &Conn{
		Id:       fmt.Sprintf("conn-%d", connId),
		conn:     conn,
		send:     make(chan *Response),
		sendRaw:  make(chan []byte),
		hub:      hub,
		channels: make(map[*Channel]bool),
	}

	connId++

	return connection
}

func (c *Conn) Close() {
	log.Printf("Closing connection %s", c.Id)
	c.conn.Close()

	for ch := range c.channels {
		ch.handleDisconnect(c)
	}

	close(c.send)
	c.hub.disconnect <- c
}

func (c *Conn) Read() {
	defer c.Close()
	log.Printf("Starting reader %s", c.Id)

	reader := wsutil.NewReader(c.conn, ws.StateServerSide)
	decoder := json.NewDecoder(reader)

	for {
		hdr, err := reader.NextFrame()

		if err != nil {
			log.Printf("Error reading client data %v", err)
			return
		}

		log.Printf("Next frame read")

		if hdr.OpCode == ws.OpClose {
			log.Printf("Closing code received %s", c.Id)
			return
		}

		var req Message

		if err := decoder.Decode(&req); err != nil {
			log.Printf("Error decoding client data %v", err)
			return
		}

		go c.hub.handleMessage(c, &req)
	}
}

func (c *Conn) Write() {
	log.Printf("Starting writer %s", c.Id)

	writer := wsutil.NewWriter(c.conn, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(writer)

	for {
		select {
		case msg, open := <-c.send:
			if !open {
				log.Printf("Closing writer %s", c.Id)
				return
			}

			if err := encoder.Encode(&msg); err != nil {
				log.Printf("Error encoding response data %v", err)
				return
			}

			log.Printf("Message decoded successfully")

			if err := writer.Flush(); err != nil {
				log.Printf("Error flushing writer %v", err)
				return
			}

		case msg, open := <-c.sendRaw:
			if !open {
				log.Printf("Closing raw writer %s", c.Id)
				return
			}

			_, err := c.conn.Write(msg)

			if err != nil {
				log.Printf("Error sending raw message %s", err)
				return
			}
		}
	}
}

func (c *Conn) Send(resp *Response) {
	c.send <- resp
}

type ConnectionMap struct {
	sync.RWMutex
	connections map[*Conn]bool

	register   chan *Conn
	unregister chan *Conn
}

func NewConnectionMap() *ConnectionMap {
	return &ConnectionMap{
		connections: make(map[*Conn]bool),
	}
}

func (cm *ConnectionMap) Add(conn *Conn) {
	cm.Lock()
	defer cm.Unlock()

	cm.connections[conn] = true
}

func (cm *ConnectionMap) Del(conn *Conn) {
	cm.Lock()
	defer cm.Unlock()

	delete(cm.connections, conn)
}

func (cm *ConnectionMap) Has(conn *Conn) bool {
	cm.Lock()
	defer cm.Unlock()

	connVal, ok := cm.connections[conn]

	return ok && connVal
}