package gosock

import (
	"bytes"
	"encoding/json"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

const (
	// Send to all connections on channel
	emitType chanMessageType = "emit"

	// Send to all connections other than sending connection
	broadcastType = "broadcast"

	// Only send to one connection
	replyType = "reply"
)

type J map[string]interface{}

type chanMessageType string

type ChannelMessage struct {
	conn     *Conn
	Type     chanMessageType `json:"type"`
	Response *Response       `json:"response"`
}

func EmitChannelMsg(conn *Conn, response *Response) *ChannelMessage {
	return &ChannelMessage{
		conn:     conn,
		Response: response,
		Type:     emitType,
	}
}

func BroadcastChannelMsg(conn *Conn, response *Response) *ChannelMessage {
	return &ChannelMessage{
		conn:     conn,
		Response: response,
		Type:     broadcastType,
	}
}

func ReplyChannelMsg(conn *Conn, response *Response) *ChannelMessage {
	return &ChannelMessage{
		conn:     conn,
		Response: response,
		Type:     replyType,
	}
}

// func (cm *ChannelMessage) Encode() ([]byte, error) {
// 	var buf bytes.Buffer
// 	w := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
// 	encoder := json.NewEncoder(w)
//
// 	if err := encoder.Encode(cm); err != nil {
// 		return nil, err
// 	}
//
// 	if err := w.Flush(); err != nil {
// 		return nil, err
// 	}
//
// 	return buf.Bytes(), nil
// }

func (cm ChannelMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(cm)
}

type Response struct {
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

func (response Response) MarshalBinary() ([]byte, error) {
	return json.Marshal(response)
}

func ResponseFromBytes(data []byte) (*Response, error) {
	var response Response

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (e *Response) Encode() ([]byte, error) {
	var buf bytes.Buffer
	w := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	if err := encoder.Encode(e); err != nil {
		return nil, err
	}

	if err := w.Flush(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
