package gateway

import (
	"encoding/json"
	"time"

	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/google/uuid"
)

const (
	FrameTypeConnect  = "connect"
	FrameTypeRequest  = "request"
	FrameTypeResponse = "response"
	FrameTypeEvent    = "event"
)

type ConnectFrame struct {
	Type      string               `json:"type"`
	Role      string               `json:"role"`
	Token     string               `json:"token,omitempty"`
	ClientID  string               `json:"client_id,omitempty"`
	Node      *node.NodeDescriptor `json:"node,omitempty"`
	Timestamp int64                `json:"timestamp"`
}

type RequestFrame struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Action    string          `json:"action"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

type ResponseFrame struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	OK        bool            `json:"ok"`
	Error     string          `json:"error,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

type EventFrame struct {
	Type      string          `json:"type"`
	Event     string          `json:"event"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

func NewConnectFrame(role, token string, descriptor *node.NodeDescriptor) ConnectFrame {
	return ConnectFrame{
		Type:      FrameTypeConnect,
		Role:      role,
		Token:     token,
		Node:      descriptor,
		Timestamp: time.Now().Unix(),
	}
}

func NewRequestFrame(action string, payload interface{}) (RequestFrame, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return RequestFrame{}, err
	}
	return RequestFrame{
		Type:      FrameTypeRequest,
		ID:        uuid.NewString(),
		Action:    action,
		Payload:   body,
		Timestamp: time.Now().Unix(),
	}, nil
}

func NewResponseFrame(id string, ok bool, payload interface{}, errText string) (ResponseFrame, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return ResponseFrame{}, err
	}
	return ResponseFrame{
		Type:      FrameTypeResponse,
		ID:        id,
		OK:        ok,
		Error:     errText,
		Payload:   body,
		Timestamp: time.Now().Unix(),
	}, nil
}

func NewEventFrame(name string, payload interface{}) (EventFrame, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return EventFrame{}, err
	}
	return EventFrame{
		Type:      FrameTypeEvent,
		Event:     name,
		Payload:   body,
		Timestamp: time.Now().Unix(),
	}, nil
}
