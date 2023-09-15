package gosock

import "encoding/json"

const JoinEventName = "__join__"
const BeforeJoinEventName = "__before_join__"

const LeaveEventName = "__leave__"
const AfterLeaveEventName = "__after_leave__"

const DisconnectEventName = "__disconnect__"

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
	return bm.Event == JoinEventName
}

type Response struct {
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

type M map[string]interface{}
