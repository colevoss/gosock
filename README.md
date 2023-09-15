# Gosock

Gosock is a channel based Websocket library built on top of [gobwas](https://github.com/gobwas/ws).
This project is in its early stages and not production ready.

## Usage

Create a Channel for users to connect to:

```go
type MyChannel struct {}

func (mc *Mychannel) Join(c *gosock.Channel, conn *gosock.Conn, msg *gosock.Message) error {
  log.Printf("User joined %s", conn.Id)

  // Send message to all channel members other than this connection
  c.Broadcast(conn, "user-joined", gosock.M{
    "id": conn.Id,
  })

  // Reply just to this connection
  c.Reply(conn, "welcome", gosock.M{
    "message": "Welcome to the channel",
  })

  return nil
}

type ChatPayload struct {
  Message string `json:"message"`
}

func (mc *MyChannel) Chat(c *gosock.Channel, conn *gosock.Conn, msg *gosock.Message) error {
  var chatPayload ChatPayload

  // Unmarshal message payload to Chat struct
  err := msg.BindPayload(&chatPayload)

  // TODO: Improve error handling
  if err != nil {
    return err
  }

  log.Printf("User %s sent chat message %s", conn.Id, chatPayload.Message)

  c.Broadcat(conn, "new-chat", gosock.M{
    "message": chatPayload.Message,
  })

  return nil
}

func main() {
  mc := &MyChannel{}
  server := gosock.NewHub()

  // Register channel for path
  // Channel paths can can contain named params in {curlyBrackets}
  server.Channel("channel.{id}.chat", func(r *Router) {
    r.On(gosock.Join(mc.Join))

    // Custom event handlers
    r.Event("new-chat", mc.Chat)
  })

  // Start server
  server.Start()
}
```

## Incoming Message

Incoming messages should be shaped like so:

```json
{
    "channel": "channel.123.chat",
    "event": "event-name",
    "payload": {
        "hello": "param"
    }
}
```

Outgoing messages are shaped the same way.

## Todo
- [ ] Fix channel names with ending params `channel.{id}`
- [ ] Add more configuration for servers
