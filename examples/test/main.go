package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/colevoss/gosock"
)

type TestRouter struct {
}

type TestPayload struct {
	Hello string `json:"hello"`
}

func (tr *TestRouter) Join(ctx context.Context, c *gosock.Channel) error {
	var tp TestPayload

	if err := gosock.BindPayload(ctx, &tp); err != nil {
		log.Printf("Badd things happened %s", err)
	}

	log.Printf("JOIN %s!!", tp.Hello)
	userId := ctx.Value("userId").(int)

	c.Broadcast(ctx, "user-joined", gosock.M{
		"id": userId,
	})

	c.Reply(ctx, "welcome", gosock.M{
		"message": fmt.Sprintf("Welcome to the channel %d", userId),
	})

	return nil
}

func (tr *TestRouter) BeforeJoin(ctx context.Context, c *gosock.Channel) error {
	log.Printf("BEFORE JOIN")

	return nil
}

func (tr *TestRouter) MyEvent(ctx context.Context, c *gosock.Channel) error {
	param, _ := c.Params.Get("param")
	userId := ctx.Value("userId").(int)
	log.Printf("%d My event received on channel %s %s", userId, c.Path, param)

	payload := struct{ Hello string }{
		Hello: "howdy",
	}

	c.Broadcast(ctx, "message", payload)

	return nil
}

func (tr *TestRouter) Disconnect(ctx context.Context, c *gosock.Channel) error {
	userId := ctx.Value("userId").(int)
	log.Printf("Disconnected from channel %d", userId)

	payload := gosock.M{
		"connId": userId,
	}

	c.Broadcast(ctx, "user-disconnect", payload)

	return nil
}

func (tr *TestRouter) Leave(ctx context.Context, c *gosock.Channel) error {
	userId := ctx.Value("userId").(int)
	log.Printf("Left channel %d", userId)

	leaveMessage := gosock.M{
		"conn-id": userId,
	}

	c.Broadcast(ctx, "user-leave", leaveMessage)

	return nil
}

func OnConnect(conn *gosock.Conn) {
	log.Printf("Connection: %s", conn.Id)
	userId := rand.Intn(100)
	log.Printf("on connect User ID: %d", userId)
	ctx := context.WithValue(conn.Context(), "userId", userId)
	// This feels gross
	conn.WithContext(ctx)
}

func main() {
	tr := &TestRouter{}
	pool := gosock.NewPool(1, 1, time.Second*60)

	server := gosock.NewHub(pool)

	server.Channel("test.{param}", func(r *gosock.Router) {
		r.On(
			gosock.Join(tr.Join),
			gosock.Leave(tr.Leave),
			gosock.Disconnect(tr.Disconnect),
		)

		r.Event("my_event", tr.MyEvent)
	})

	server.On(gosock.Connect(OnConnect))

	server.Start()
}
