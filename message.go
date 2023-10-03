package gosock

import (
	"encoding/json"
)

const (
	joinEventName       = "__join__"
	beforeJoinEventName = "__before_join__"
	leaveEventName      = "__leave__"
	afterLeaveEventName = "__after_leave__"
	disconnectEventName = "__disconnect__"
)

type Message struct {
	Channel string          `json:"channel"`
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func (m *Message) RawPayload() (map[string]interface{}, error) {
	var p map[string]interface{}
	err := json.Unmarshal(m.Payload, &p)

	return p, err
}

func (m *Message) BindPayload(p interface{}) error {
	return json.Unmarshal(m.Payload, p)
}

func (m Message) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func MessageFromBytes(data []byte) (*Message, error) {
	var message Message

	if err := json.Unmarshal(data, &message); err != nil {
		return nil, err
	}

	return &message, nil
}
