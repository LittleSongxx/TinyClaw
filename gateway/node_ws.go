package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/gorilla/websocket"
)

var websocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type responseWaiter struct {
	result *node.NodeCommandResult
	err    error
}

type websocketNodeTransport struct {
	conn *websocket.Conn

	writeMu sync.Mutex
	waitMu  sync.Mutex
	waiters map[string]chan responseWaiter
	closed  chan struct{}
	onEvent func(EventFrame)
	onClose func()
}

func (s *Service) HandleNodeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	var connect ConnectFrame
	if err := conn.ReadJSON(&connect); err != nil {
		_ = conn.Close()
		return
	}
	if connect.Type != FrameTypeConnect || connect.Role != "node" || connect.Node == nil {
		_ = conn.Close()
		return
	}
	if connect.ProtocolVersion != ProtocolVersionV1 {
		_ = conn.Close()
		return
	}
	if connect.Auth.Type == "bootstrap" {
		publicKey := firstNonEmpty(connect.Auth.PublicKey, devicePublicKey(connect))
		request, pairErr := s.SubmitDevicePairing(r.Context(), PairingSubmitRequest{
			BootstrapCode: firstNonEmpty(connect.Auth.Token, connect.Token),
			DeviceID:      connect.DeviceID(),
			PublicKey:     publicKey,
			Descriptor:    connect.Node,
		})
		if pairErr == nil {
			frame, _ := NewEventFrame("devices.pairing_pending", request)
			_ = conn.WriteJSON(frame)
		}
		_ = conn.Close()
		return
	}
	device, err := s.VerifyDeviceConnect(r.Context(), connect)
	if err != nil || device == nil {
		_ = conn.Close()
		return
	}
	connect.Node.WorkspaceID = device.WorkspaceID
	connect.Node.DeviceID = device.DeviceID

	transport := &websocketNodeTransport{
		conn:    conn,
		waiters: make(map[string]chan responseWaiter),
		closed:  make(chan struct{}),
		onClose: func() {
			if s != nil && s.nodes != nil {
				s.nodes.RemoveNode(connect.Node.ID)
			}
		},
		onEvent: func(frame EventFrame) {
			if frame.Event == "node.heartbeat" {
				s.nodes.Heartbeat(connect.Node.ID)
			}
		},
	}

	if err := s.nodes.RegisterNode(r.Context(), *connect.Node, transport); err != nil {
		_ = conn.Close()
		return
	}

	go transport.readLoop(connect.Node.ID)
}

func devicePublicKey(connect ConnectFrame) string {
	if connect.Device != nil && connect.Device.PublicKey != "" {
		return connect.Device.PublicKey
	}
	if connect.Node != nil && len(connect.Node.Metadata) > 0 {
		return connect.Node.Metadata["public_key"]
	}
	return ""
}

func (t *websocketNodeTransport) Request(ctx context.Context, req node.NodeCommandRequest) (*node.NodeCommandResult, error) {
	frame, err := NewRequestFrame("node.command", req)
	if err != nil {
		return nil, err
	}

	waiter := make(chan responseWaiter, 1)
	t.waitMu.Lock()
	t.waiters[frame.ID] = waiter
	t.waitMu.Unlock()

	if err := t.writeJSON(frame); err != nil {
		t.waitMu.Lock()
		delete(t.waiters, frame.ID)
		t.waitMu.Unlock()
		return nil, err
	}

	select {
	case reply := <-waiter:
		return reply.result, reply.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.closed:
		return nil, errors.New("node connection closed")
	}
}

func (t *websocketNodeTransport) Close() error {
	select {
	case <-t.closed:
	default:
		close(t.closed)
	}
	return t.conn.Close()
}

func (t *websocketNodeTransport) readLoop(nodeID string) {
	defer func() {
		_ = t.Close()
		if t.onClose != nil {
			t.onClose()
		}
	}()

	for {
		var raw map[string]json.RawMessage
		if err := t.conn.ReadJSON(&raw); err != nil {
			return
		}

		var frameType string
		_ = json.Unmarshal(raw["type"], &frameType)

		switch frameType {
		case FrameTypeResponse, legacyFrameTypeResponse:
			var response ResponseFrame
			if err := decodeRawFrame(raw, &response); err != nil {
				continue
			}

			var result node.NodeCommandResult
			if body := response.RawResult(); len(body) > 0 {
				_ = json.Unmarshal(body, &result)
			}
			if result.NodeID == "" {
				result.NodeID = nodeID
			}
			if result.ID == "" {
				result.ID = response.ID
			}

			t.waitMu.Lock()
			waiter := t.waiters[response.ID]
			delete(t.waiters, response.ID)
			t.waitMu.Unlock()
			if waiter != nil {
				if response.OK {
					waiter <- responseWaiter{result: &result}
				} else {
					waiter <- responseWaiter{result: &result, err: errors.New(response.ErrorMessage())}
				}
			}
		case FrameTypeEvent:
			var event EventFrame
			if err := decodeRawFrame(raw, &event); err != nil {
				continue
			}
			if t.onEvent != nil {
				t.onEvent(event)
			}
		}
	}
}

func (t *websocketNodeTransport) writeJSON(payload interface{}) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	return t.conn.WriteJSON(payload)
}

func decodeRawFrame(raw map[string]json.RawMessage, target interface{}) error {
	body, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}
