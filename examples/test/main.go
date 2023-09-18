package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
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
		"id":  userId,
		"msg": tp.Hello,
	})

	c.Reply(ctx, "welcome", gosock.M{
		"id":      userId,
		"message": fmt.Sprintf("Welcome to the channel %d", userId),
	})

	return nil
}

func (tr *TestRouter) BeforeJoin(ctx context.Context, c *gosock.Channel) error {
	log.Printf("BEFORE JOIN")

	return nil
}

type Message struct {
	Msg string `json:"message"`
}

func (tr *TestRouter) MyEvent(ctx context.Context, c *gosock.Channel) error {
	param, _ := c.Param("param")
	userId := ctx.Value("userId").(int)
	log.Printf("%d My event received on channel %s (%s)", userId, c.Path(), param)

	var msg Message

	if err := gosock.BindPayload(ctx, &msg); err != nil {
		log.Printf("FUCKING PAYLOAD %s", err)
	}

	c.Broadcast(ctx, "message", gosock.M{
		"msg":    msg.Msg,
		"userId": userId,
	})

	// c.Reply(ctx, "howdy", gosock.M{
	// 	"yes": msg.Hello,
	// })

	return nil
}

func (tr *TestRouter) Disconnect(ctx context.Context, c *gosock.Channel) error {
	userId := ctx.Value("userId").(int)
	log.Printf("Disconnected from channel %d", userId)

	payload := gosock.M{
		"id": userId,
	}

	c.Emit("user-disconnect", payload)

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

func UserIdMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("MY MIDDLEWARE!!!!!!!!!!")
		userId := rand.Intn(100)

		log.Printf("on connect User ID: %d", userId)
		ctx := context.WithValue(r.Context(), "userId", userId)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	tr := &TestRouter{}
	pool := gosock.NewPool(1, 1, time.Second*60)

	server := gosock.NewHub(pool)

	server.Use(UserIdMiddleware)

	server.Channel("test.{param}", func(r *gosock.Router) {
		r.On(
			r.Join(tr.Join),
			r.Leave(tr.Leave),
			r.Disconnect(tr.Disconnect),
		)

		r.Event("my_event", tr.MyEvent)
	})

	server.Start()
	http.ListenAndServe(":8080", server)
}
