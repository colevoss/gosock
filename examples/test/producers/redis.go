package producers

import (
	"context"
	"log"

	"github.com/colevoss/gosock"
	"github.com/redis/go-redis/v9"
)

type RedisManager struct {
	rdb *redis.Client
}

func (rm *RedisManager) Connect() {
	addr := "localhost:6379"
	log.Printf("Connecting to redis cluster %s", addr)

	rm.rdb = redis.NewClient(&redis.Options{
		Addr:      addr,
		Password:  "",
		DB:        0,
		OnConnect: rm.onConnect,
	})
}

func (rm *RedisManager) onConnect(ctx context.Context, conn *redis.Conn) error {
	log.Printf("Connected to redis")
	return nil
}

func (rm *RedisManager) Create(channel *gosock.Channel) gosock.Producer {
	return NewReddisProducer(rm, channel)
}

type RedisProducer struct {
	manager *RedisManager
	channel *gosock.Channel
	pubsub  *redis.PubSub
}

func NewReddisProducer(manager *RedisManager, channel *gosock.Channel) *RedisProducer {
	return &RedisProducer{
		manager: manager,
		channel: channel,
	}
}

func (rp *RedisProducer) Subscribe() {
	ctx := context.Background()
	path := rp.channel.Path()

	rp.pubsub = rp.manager.rdb.Subscribe(ctx, path)

	go rp.subscribe()
}

func (rp *RedisProducer) Stop() {
	rp.pubsub.Close()
}

func (rp *RedisProducer) subscribe() {
	pubSubChannel := rp.pubsub.Channel()

	for msg := range pubSubChannel {
		log.Printf("Redis message: %s %s", msg.Channel, msg.Payload)

		resp, err := gosock.ResponseFromBytes([]byte(msg.Payload))

		if err != nil {
			log.Printf("Error unmarshalling data %s", err)
			continue
		}
		rp.channel.SendResp(resp)
	}
}

func (rp *RedisProducer) Publish(ctx context.Context, response *gosock.Response) {
	pub := rp.manager.rdb.Publish(ctx, response.Channel, response)

	err := pub.Err()

	if err != nil {
		log.Printf("Error sending msg to redis channel %+v %s", response, err)
	}
}
