# Gosock

Gosock is a channel based Websocket library built on top of [gobwas](https://github.com/gobwas/ws).
This project is in its early stages and not production ready.

## Usage

See [Example](./examples/test/main.go)

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
- [x] Fix channel names with ending params `channel.{id}`
- [ ] Add more configuration for servers
