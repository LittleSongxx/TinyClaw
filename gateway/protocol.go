package gateway

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/google/uuid"
)

const (
	ProtocolVersionV1 = "v1"

	FrameTypeConnect  = "connect"
	FrameTypeRequest  = "req"
	FrameTypeResponse = "res"
	FrameTypeEvent    = "event"

	legacyFrameTypeRequest  = "request"
	legacyFrameTypeResponse = "response"
)

type AuthInfo struct {
	Type       string `json:"type"`
	Token      string `json:"token,omitempty"`
	DeviceID   string `json:"device_id,omitempty"`
	Nonce      string `json:"nonce,omitempty"`
	Signature  string `json:"signature,omitempty"`
	PublicKey  string `json:"public_key,omitempty"`
	ActorToken string `json:"actor_token,omitempty"`
}

type DeviceInfo struct {
	ID        string            `json:"id,omitempty"`
	Name      string            `json:"name,omitempty"`
	Platform  string            `json:"platform,omitempty"`
	PublicKey string            `json:"public_key,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type ConnectFrame struct {
	Type            string               `json:"type"`
	ProtocolVersion string               `json:"protocol_version"`
	Role            string               `json:"role"`
	Auth            AuthInfo             `json:"auth"`
	WorkspaceID     string               `json:"workspace_id,omitempty"`
	Token           string               `json:"token,omitempty"`
	ClientID        string               `json:"client_id,omitempty"`
	Node            *node.NodeDescriptor `json:"node,omitempty"`
	Device          *DeviceInfo          `json:"device,omitempty"`
	Timestamp       int64                `json:"timestamp"`
}

type RequestFrame struct {
	Type           string          `json:"type"`
	ID             string          `json:"id"`
	Method         string          `json:"method"`
	Params         json.RawMessage `json:"params,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	Action         string          `json:"action,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	WorkspaceID    string          `json:"workspace_id,omitempty"`
	Timestamp      int64           `json:"timestamp"`
}

type FrameError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ResponseFrame struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	OK        bool            `json:"ok"`
	Error     *FrameError     `json:"error,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

type EventFrame struct {
	Type         string          `json:"type"`
	Event        string          `json:"event"`
	Seq          int64           `json:"seq,omitempty"`
	StateVersion int64           `json:"state_version,omitempty"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	Timestamp    int64           `json:"timestamp"`
}

func NewConnectFrame(role, token string, descriptor *node.NodeDescriptor) ConnectFrame {
	return ConnectFrame{
		Type:            FrameTypeConnect,
		ProtocolVersion: ProtocolVersionV1,
		Role:            role,
		Auth: AuthInfo{
			Type:  "bearer",
			Token: token,
		},
		Token:     token,
		Node:      descriptor,
		Timestamp: time.Now().Unix(),
	}
}

func NewRequestFrame(method string, params interface{}) (RequestFrame, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return RequestFrame{}, err
	}
	return RequestFrame{
		Type:      FrameTypeRequest,
		ID:        uuid.NewString(),
		Method:    method,
		Params:    body,
		Timestamp: time.Now().Unix(),
	}, nil
}

func NewResponseFrame(id string, ok bool, result interface{}, errText string) (ResponseFrame, error) {
	body, err := json.Marshal(result)
	if err != nil {
		return ResponseFrame{}, err
	}
	var frameErr *FrameError
	if errText != "" {
		frameErr = &FrameError{Code: "error", Message: errText}
	}
	return ResponseFrame{
		Type:      FrameTypeResponse,
		ID:        id,
		OK:        ok,
		Error:     frameErr,
		Result:    body,
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

func (f RequestFrame) MethodName() string {
	if f.Method != "" {
		return f.Method
	}
	return f.Action
}

func (f RequestFrame) RawParams() json.RawMessage {
	if len(f.Params) > 0 {
		return f.Params
	}
	return f.Payload
}

func (f RequestFrame) Validate() error {
	if f.Type != FrameTypeRequest && f.Type != legacyFrameTypeRequest {
		return errors.New("request frame type must be req")
	}
	if f.ID == "" {
		return errors.New("request id is required")
	}
	if f.MethodName() == "" {
		return errors.New("request method is required")
	}
	if requiresIdempotency(f.MethodName()) && f.IdempotencyKey == "" {
		return errors.New("idempotency_key is required for side-effect method")
	}
	return nil
}

func (f ResponseFrame) RawResult() json.RawMessage {
	if len(f.Result) > 0 {
		return f.Result
	}
	return f.Payload
}

func (f ResponseFrame) ErrorMessage() string {
	if f.Error == nil {
		return ""
	}
	return f.Error.Message
}

func requiresIdempotency(method string) bool {
	switch method {
	case "node.command", "node.exec", "approvals.decide", "flows.run", "devices.approve", "devices.reject", "devices.revoke", "plugins.enable", "plugins.disable":
		return true
	default:
		return false
	}
}

func isResponseFrameType(frameType string) bool {
	return frameType == FrameTypeResponse || frameType == legacyFrameTypeResponse
}
