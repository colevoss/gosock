package gosock

import "encoding/json"

const Join = "__join__"
const BeforeJoin = "__before_join__"

const Leave = "__leave__"
const AfterLeave = "__after_leave__"

const Disconnect = "__disconnect__"

type Message struct {
	Channel string `json:"channel"`
	Event   string `json:"event"`
	// Payload map[string]interface{} `json:"payload"`
	Payload json.RawMessage `json:"payload"`
}

func (bm *Message) RawPayload() (map[string]interface{}, error) {
	var p map[string]interface{}
	err := json.Unmarshal(bm.Payload, &p)

	return p, err
}

func (bm *Message) BindPayload(p interface{}) error {
	return json.Unmarshal(bm.Payload, p)
}

func (bm *Message) IsJsoin() bool {
	return bm.Event == Join
}

type Response struct {
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

type M map[string]interface{}
