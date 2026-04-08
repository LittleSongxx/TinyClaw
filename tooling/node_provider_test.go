package tooling

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/LittleSongxx/TinyClaw/node"
)

type fakeNodeTransport struct {
	result *node.NodeCommandResult
}

func (f *fakeNodeTransport) Request(ctx context.Context, req node.NodeCommandRequest) (*node.NodeCommandResult, error) {
	if f.result == nil {
		return &node.NodeCommandResult{
			ID:         req.ID,
			NodeID:     req.NodeID,
			Capability: req.Capability,
			Success:    true,
		}, nil
	}
	result := *f.result
	result.ID = req.ID
	result.NodeID = req.NodeID
	result.Capability = req.Capability
	return &result, nil
}

func (f *fakeNodeTransport) Close() error { return nil }

func TestNodeProviderListsConnectedCapabilities(t *testing.T) {
	manager := node.NewManager()
	err := manager.RegisterNode(context.Background(), node.NodeDescriptor{
		ID: "node-1",
		Capabilities: []node.NodeCapability{
			{Name: "system.exec"},
			{Name: "screen.snapshot"},
			{Name: "input.keyboard.type"},
			{Name: "ui.find"},
			{Name: "ui.focus"},
		},
	}, &fakeNodeTransport{})
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	provider := NewNodeProvider(manager)
	specs, err := provider.ListTools(context.Background())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	names := make(map[string]bool, len(specs))
	for _, spec := range specs {
		names[spec.Name] = true
	}

	for _, name := range []string{toolNodeListDevices, toolNodeSystemExec, toolNodeScreenShot, toolNodeKeyboardType, toolNodeUIFind, toolNodeUIFocus} {
		if !names[name] {
			t.Fatalf("expected tool %s to be exposed, got %+v", name, specs)
		}
	}
}

func TestNodeProviderFormatsScreenshotAsImagePayload(t *testing.T) {
	manager := node.NewManager()
	err := manager.RegisterNode(context.Background(), node.NodeDescriptor{
		ID: "node-1",
		Capabilities: []node.NodeCapability{
			{Name: "screen.snapshot"},
		},
	}, &fakeNodeTransport{
		result: &node.NodeCommandResult{
			Success: true,
			Data: map[string]interface{}{
				"mime_type": "image/png",
				"base64":    "ZmFrZS1pbWFnZQ==",
			},
		},
	})
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	provider := NewNodeProvider(manager)
	result, err := provider.ExecuteTool(context.Background(), ToolInvocation{
		Name: toolNodeScreenShot,
		Arguments: map[string]interface{}{
			argNodeID: "node-1",
		},
	})
	if err != nil {
		t.Fatalf("execute tool: %v", err)
	}
	if !strings.Contains(result.Output, `"type":"image"`) || !strings.Contains(result.Output, `ZmFrZS1pbWFnZQ==`) {
		t.Fatalf("expected image payload, got %s", result.Output)
	}
}

func TestNodeProviderReturnsPendingApprovalPayload(t *testing.T) {
	manager := node.NewManager()
	err := manager.RegisterNode(context.Background(), node.NodeDescriptor{
		ID: "node-1",
		Capabilities: []node.NodeCapability{
			{Name: "input.keyboard.type"},
		},
	}, &fakeNodeTransport{})
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	provider := NewNodeProvider(manager)
	result, err := provider.ExecuteTool(context.Background(), ToolInvocation{
		Name: toolNodeKeyboardType,
		Arguments: map[string]interface{}{
			argNodeID: "node-1",
			"text":    "hello",
		},
	})
	if err != nil {
		t.Fatalf("execute tool: %v", err)
	}
	if !strings.Contains(result.Output, `"pending_approval":true`) {
		t.Fatalf("expected pending approval payload, got %s", result.Output)
	}
	if !strings.Contains(result.Output, `"approval_id"`) {
		t.Fatalf("expected approval id in payload, got %s", result.Output)
	}
}

func TestBuildNodeToolSpecSupportsActiveWindowAndElementLocators(t *testing.T) {
	screenshotSpec, ok := buildNodeToolSpec("screen.snapshot")
	if !ok {
		t.Fatal("expected screen.snapshot tool spec")
	}
	screenJSON, err := json.Marshal(screenshotSpec.InputSchema)
	if err != nil {
		t.Fatalf("marshal screenshot schema: %v", err)
	}
	if !strings.Contains(string(screenJSON), "active_window") || !strings.Contains(string(screenJSON), "window_handle") {
		t.Fatalf("expected active_window selectors in screenshot schema, got %s", string(screenJSON))
	}

	findSpec, ok := buildNodeToolSpec("ui.find")
	if !ok {
		t.Fatal("expected ui.find tool spec")
	}
	findJSON, err := json.Marshal(findSpec.InputSchema)
	if err != nil {
		t.Fatalf("marshal ui.find schema: %v", err)
	}
	for _, key := range []string{"automation_id", "class_name", "path", "max_results"} {
		if !strings.Contains(string(findJSON), key) {
			t.Fatalf("expected %s in ui.find schema, got %s", key, string(findJSON))
		}
	}
}
