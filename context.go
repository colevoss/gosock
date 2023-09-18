package gosock

import (
	"context"
	"encoding/json"
)

type ctxKey string

func withMessage(ctx context.Context, msg *Message) context.Context {
	return context.WithValue(ctx, ctxKey("msg"), msg)
}

func withConnection(ctx context.Context, conn *Conn) context.Context {
	return context.WithValue(ctx, ctxKey("conn"), conn)
}

func withParams(ctx context.Context, params *Params) context.Context {
	return context.WithValue(ctx, ctxKey("params"), params)
}

func GetParams(ctx context.Context) *Params {
	params := ctx.Value(ctxKey("params"))

	if params == nil {
		return nil
	}

	return params.(*Params)
}

func Param(ctx context.Context, key string) (string, bool) {
	params := GetParams(ctx)
	if params == nil {
		return "", false
	}

	return params.Get(key)
}

func GetConnection(ctx context.Context) *Conn {
	conn := ctx.Value(ctxKey("conn"))

	if conn == nil {
		return nil
	}

	return conn.(*Conn)
}

func BindPayload(ctx context.Context, p interface{}) error {
	msg := ctx.Value(ctxKey("msg")).(*Message)

	return json.Unmarshal(msg.Payload, p)
}
