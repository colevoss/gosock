package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/colevoss/gosock"
	"github.com/colevoss/gosock/examples/test/chat"
	"github.com/colevoss/gosock/examples/test/db"
	"github.com/colevoss/gosock/examples/test/producers"
)

var (
	addr = flag.String("port", getEnv("PORT", "8080"), "Port")
)

func main() {
	db := db.NewDb()
	middleware := chat.NewMiddleware(db)
	chatRouter := chat.NewChatRouter(db)

	pool := gosock.NewPool(10, 10, time.Second*60)

	server := gosock.NewHub(pool)
	redisManager := &producers.RedisManager{}
	redisManager.Connect()
	server.AddProducerManager(redisManager)

	server.Use(middleware.UserMiddleware)

	server.Channel("chat.{channelId}", func(r *gosock.Router) {
		r.On(
			r.Join(chatRouter.Join),
			r.Disconnect(chatRouter.Disconnected),
		)

		r.Event("chat", chatRouter.Chat)
	})

	server.Start()
	serverPort := fmt.Sprintf(":%s", *addr)
	log.Printf("Starting server at addr %s", serverPort)
	http.ListenAndServe(serverPort, server)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return fallback
}
