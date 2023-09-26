package gosock

import "encoding/json"

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

func (bm *Message) RawPayload() (map[string]interface{}, error) {
	var p map[string]interface{}
	err := json.Unmarshal(bm.Payload, &p)

	return p, err
}

func (bm *Message) BindPayload(p interface{}) error {
	return json.Unmarshal(bm.Payload, p)
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

type M map[string]interface{}
