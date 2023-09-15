package main

import (
	"fmt"
	"log"
	"time"

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
		"message": fmt.Sprintf("Welcome to the channel %s", conn.Id),
	})

	return nil
}

func (tr *TestRouter) BeforeJoin(c *gosock.Channel, conn *gosock.Conn, msg *gosock.Message) error {
	log.Printf("BEFORE JOIN")

	return nil
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
	pool := gosock.NewPool(1, 1, time.Second*60)

	pool.Handle(gosock.PoolOpen(func(id int) {
		log.Printf("Opening worker pool: %d", id)
	}))

	pool.Handle(gosock.PoolClose(func(id int) {
		log.Printf("Closing worker pool: %d", id)
	}))
	server := gosock.NewHub(pool)

	server.Channel("test.{param}", func(r *gosock.Router) {
		r.On(
			gosock.Join(tr.Join),
			gosock.Leave(tr.Leave),
			gosock.Disconnect(tr.Disconnect),
		)

		r.Event("my_event", tr.MyEvent)
	})

	server.On(gosock.Connect, OnConnect)

	server.Start()
}
