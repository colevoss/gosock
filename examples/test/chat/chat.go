package chat

import (
	"context"
	"log"

	"github.com/colevoss/gosock"
	"github.com/colevoss/gosock/examples/test/db"
)

type ChatRouter struct {
	db *db.Db
}

func NewChatRouter(db *db.Db) *ChatRouter {
	return &ChatRouter{db}
}

func (tr *ChatRouter) Join(ctx context.Context, c *gosock.Channel) error {
	userId, _ := UserId(ctx)

	// c.Emit(ctx, "balls", gosock.J{
	// 	"hello":  "Test",
	// 	"userId": userId,
	// })

	c.Broadcast(ctx, "user-joined", gosock.J{
		"userId": userId,
	})

	c.Reply(ctx, "welcome", gosock.J{
		"userId": userId,
	})

	return nil
}

type IncomingChatPayload struct {
	Message string `json:"message"`
}

func (cr *ChatRouter) Chat(ctx context.Context, c *gosock.Channel) error {
	userId, _ := UserId(ctx)

	var msg IncomingChatPayload

	if err := gosock.BindPayload(ctx, &msg); err != nil {
		log.Printf("Malformed payload %s", err)
	}

	c.Broadcast(ctx, "message", gosock.J{
		"message": msg.Message,
		"userId":  userId,
	})

	return nil
}

func (cr *ChatRouter) Disconnected(ctx context.Context, c *gosock.Channel) error {
	userId, _ := UserId(ctx)

	// c.Emit("user-disconnected", gosock.M{
	// 	"userId": userId,
	// })

	c.Emit(ctx, "user-disconnected", gosock.J{
		"userId": userId,
	})

	return nil
}

func (cr *ChatRouter) Register(r *gosock.Router) {
	r.On(
		r.Join(cr.Join),
		r.Disconnect(cr.Disconnected),
	)

	r.Event("chat", cr.Chat)
}
