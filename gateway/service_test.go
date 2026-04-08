package gateway

import (
	"context"
	"path/filepath"
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
	})
	if err != nil {
		t.Fatalf("execute node command: %v", err)
	}
	if !result.Success || result.Output != "ok" {
		t.Fatalf("unexpected node command result: %+v", result)
	}
}
