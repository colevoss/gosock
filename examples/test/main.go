package main

import (
	"log"

	"github.com/colevoss/gosock"
)

type TestRouter struct {
}

type TestPayload struct {
	Hello string `json:"hello"`
}

func (tr *TestRouter) Join(c *gosock.Channel, conn *gosock.Conn, msg *gosock.Message) error {
	var tp TestPayload

	err := msg.BindPayload(&tp)

	if err != nil {
		log.Printf("Badd things happened %s", err)
	}

	log.Printf("JOIN %s", tp.Hello)

	c.Broadcast(conn, "user-joined", gosock.M{
		"id": conn.Id,
	})

	c.Reply(conn, "welcome", gosock.M{
		"message": "Welcome to the channel",
	})

	return nil
}

func (tr *TestRouter) BeforeJoin(c *gosock.Channel, conn *gosock.Conn, msg *gosock.Message) error {
	log.Printf("BEFORE JOIN")

	// return fmt.Errorf("BAD THINGS %s", "yeah")

	return nil
}

func (tr *TestRouter) Register(c *gosock.Router) {
	c.Event(gosock.BeforeJoin, tr.BeforeJoin)
	c.Event(gosock.Join, tr.Join)
	c.Event(gosock.Leave, tr.Leave)
	c.Event(gosock.Disconnect, tr.Disconnect)
	c.Event("my_event", tr.MyEvent)
}

func (tr *TestRouter) MyEvent(c *gosock.Channel, conn *gosock.Conn, msg *gosock.Message) error {
	param, _ := c.Params.Get("param")
	log.Printf("%s My event received on channel %s %s", conn.Id, c.Path, param)

	payload := struct{ Hello string }{
		Hello: "howdy",
	}

	c.Broadcast(conn, "message", payload)

	return nil
}

func (tr *TestRouter) Disconnect(c *gosock.Channel, conn *gosock.Conn, msg *gosock.Message) error {
	log.Printf("Disconnected from channel %s", conn.Id)

	payload := gosock.M{
		"connId": conn.Id,
	}

	c.Broadcast(conn, "user-disconnect", payload)

	return nil
}

func (tr *TestRouter) Leave(c *gosock.Channel, conn *gosock.Conn, msg *gosock.Message) error {
	log.Printf("Left channel %s", conn.Id)
	leaveMessage := gosock.M{
		"conn-id": conn.Id,
	}

	c.Broadcast(conn, "user-leave", leaveMessage)

	return nil
}

func OnConnect(conn *gosock.Conn) {
	log.Printf("Connection: %s", conn.Id)
}

func main() {
	tr := &TestRouter{}
	server := gosock.NewHub()
	server.Channel("test.{param}.hello", tr)

	server.On(gosock.Connect, OnConnect)

	server.Start()
}
