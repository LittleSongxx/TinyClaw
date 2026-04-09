package gateway

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LittleSongxx/TinyClaw/agent"
	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/LittleSongxx/TinyClaw/session"
	"github.com/LittleSongxx/TinyClaw/tooling"
)

type fakeTransport struct {
	result *node.NodeCommandResult
}

func (f *fakeTransport) Request(ctx context.Context, req node.NodeCommandRequest) (*node.NodeCommandResult, error) {
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

func (f *fakeTransport) Close() error { return nil }

func TestBeginInboundCreatesStableDMSession(t *testing.T) {
	store := session.NewFileStore(filepath.Join(t.TempDir(), "sessions"))
	manager := node.NewManager()
	runtimeCfg := &conf.RuntimeConfig{
		Nodes: conf.NodeRuntimeConf{DefaultCommandTimeoutSec: 10},
	}
	service := NewService(runtimeCfg, store, tooling.NewBroker(), manager, agent.NewRuntime(nil, nil, nil, nil), nil)

	firstEnv, firstState, err := service.BeginInbound(context.Background(), InboundMessage{
		Channel:   "web",
		AccountID: "default",
		PeerID:    "u-1",
		MessageID: "m-1",
	})
	if err != nil {
		t.Fatalf("begin inbound first message: %v", err)
	}
	secondEnv, secondState, err := service.BeginInbound(context.Background(), InboundMessage{
		Channel:   "web",
		AccountID: "default",
		PeerID:    "u-1",
		MessageID: "m-2",
	})
	if err != nil {
		t.Fatalf("begin inbound second message: %v", err)
	}

	if firstEnv.SessionID != secondEnv.SessionID {
		t.Fatalf("expected stable dm session, got %s and %s", firstEnv.SessionID, secondEnv.SessionID)
	}
	if firstState.UseRecord || secondState.UseRecord {
		t.Fatalf("gateway session state should disable legacy record context")
	}
}

func TestExecuteNodeCommandUsesRegisteredNode(t *testing.T) {
	manager := node.NewManager()
	service := NewService(&conf.RuntimeConfig{
		Nodes: conf.NodeRuntimeConf{DefaultCommandTimeoutSec: 5},
	}, session.NewFileStore(filepath.Join(t.TempDir(), "sessions")), tooling.NewBroker(), manager, nil, nil)

	err := manager.RegisterNode(context.Background(), node.NodeDescriptor{
		ID: "node-1",
		Capabilities: []node.NodeCapability{
			{Name: "system.exec"},
		},
	}, &fakeTransport{
		result: &node.NodeCommandResult{
			Success: true,
			Output:  "ok",
		},
	})
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	result, err := service.ExecuteNodeCommand(context.Background(), node.NodeCommandRequest{
		NodeID:     "node-1",
		Capability: "system.exec",
		Arguments: map[string]interface{}{
			"command": "git",
			"args":    []interface{}{"status"},
		},
	})
	if err != nil {
		t.Fatalf("execute node command: %v", err)
	}
	if !result.Success || result.Output != "ok" {
		t.Fatalf("unexpected node command result: %+v", result)
	}
}

func TestNodeActionEventsAreWrittenToTranscript(t *testing.T) {
	store := session.NewFileStore(filepath.Join(t.TempDir(), "sessions"))
	manager := node.NewManager()
	service := NewService(&conf.RuntimeConfig{
		Nodes: conf.NodeRuntimeConf{DefaultCommandTimeoutSec: 5},
	}, store, tooling.NewBroker(), manager, nil, nil)

	err := manager.RegisterNode(context.Background(), node.NodeDescriptor{
		ID: "node-1",
		Capabilities: []node.NodeCapability{
			{Name: "fs.read"},
		},
	}, &fakeTransport{
		result: &node.NodeCommandResult{
			Success: true,
			Output:  "file body",
		},
	})
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	env, _, err := service.BeginInbound(context.Background(), InboundMessage{
		Channel:   "lark",
		AccountID: "app-1",
		PeerID:    "u-1",
		MessageID: "m-1",
	})
	if err != nil {
		t.Fatalf("begin inbound: %v", err)
	}

	_, err = service.ExecuteNodeCommand(context.Background(), node.NodeCommandRequest{
		NodeID:          "node-1",
		SessionID:       env.SessionID,
		UserID:          "u-1",
		Capability:      "fs.read",
		Arguments:       map[string]interface{}{"path": "README.md"},
		RequireApproval: true,
	})
	if err != nil {
		t.Fatalf("execute node command: %v", err)
	}

	items, err := store.Recent(context.Background(), env.SessionID, 10)
	if err != nil {
		t.Fatalf("load transcript: %v", err)
	}
	if len(items) < 3 {
		t.Fatalf("expected action events in transcript, got %+v", items)
	}
	foundRequested := false
	foundCompleted := false
	for _, item := range items {
		if item.Role != session.RoleSystem || item.Metadata["kind"] != "node_action_event" {
			continue
		}
		if strings.Contains(item.Content, `"type":"action.requested"`) {
			foundRequested = true
		}
		if strings.Contains(item.Content, `"type":"action.completed"`) {
			foundCompleted = true
		}
	}
	if !foundRequested || !foundCompleted {
		t.Fatalf("expected requested and completed action events, got %+v", items)
	}
}
