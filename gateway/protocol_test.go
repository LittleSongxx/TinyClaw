package gateway

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRequestFrameValidationRequiresIdempotencyForSideEffects(t *testing.T) {
	frame := RequestFrame{Type: FrameTypeRequest, ID: "1", Method: "node.command"}
	if err := frame.Validate(); err == nil || !strings.Contains(err.Error(), "idempotency_key") {
		t.Fatalf("expected idempotency error, got %v", err)
	}
	frame.IdempotencyKey = "key-1"
	if err := frame.Validate(); err != nil {
		t.Fatalf("expected valid side-effect frame, got %v", err)
	}
}

func TestRequestFrameSupportsLegacyActionPayload(t *testing.T) {
	frame := RequestFrame{Type: legacyFrameTypeRequest, ID: "1", Action: "nodes.list", Payload: json.RawMessage(`{"ok":true}`)}
	if frame.MethodName() != "nodes.list" {
		t.Fatalf("expected legacy action method name, got %q", frame.MethodName())
	}
	if string(frame.RawParams()) != `{"ok":true}` {
		t.Fatalf("expected legacy payload params, got %s", string(frame.RawParams()))
	}
}

func TestProtocolSchemaContainsV1Frames(t *testing.T) {
	body, err := ProtocolJSONSchemaBytes()
	if err != nil {
		t.Fatalf("schema marshal: %v", err)
	}
	text := string(body)
	for _, want := range []string{`"connect"`, `"req"`, `"res"`, `"event"`, `"protocol_version"`, `"idempotency_key"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("schema missing %s: %s", want, text)
		}
	}
}
