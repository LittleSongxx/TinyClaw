package node

import (
	"context"
	"strings"
	"testing"
	"time"
)

type managerTestTransport struct {
	lastReq NodeCommandRequest
}

func (m *managerTestTransport) Request(ctx context.Context, req NodeCommandRequest) (*NodeCommandResult, error) {
	m.lastReq = req
	return &NodeCommandResult{
		ID:         req.ID,
		NodeID:     req.NodeID,
		Capability: req.Capability,
		Success:    true,
		Output:     "ok",
	}, nil
}

func (m *managerTestTransport) Close() error { return nil }

func TestManagerApprovalFlow(t *testing.T) {
	manager := NewManager()
	transport := &managerTestTransport{}
	err := manager.RegisterNode(context.Background(), NodeDescriptor{
		ID: "node-1",
		Capabilities: []NodeCapability{
			{Name: "input.keyboard.type"},
		},
	}, transport)
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	result, err := manager.Execute(context.Background(), NodeCommandRequest{
		ID:              "approval-1",
		UserID:          "user-1",
		SessionID:       "session-1",
		Capability:      "input.keyboard.type",
		Arguments:       map[string]interface{}{"text": "hello"},
		RequireApproval: true,
	})
	if err != nil {
		t.Fatalf("execute with approval: %v", err)
	}
	if pending, _ := result.Data["pending_approval"].(bool); !pending {
		t.Fatalf("expected pending approval response, got %+v", result)
	}

	items := manager.ListApprovals(context.Background())
	if len(items) != 1 || items[0].ID != "approval-1" {
		t.Fatalf("expected one approval, got %+v", items)
	}
	if len(items[0].ApprovalModes) != 1 || items[0].ApprovalModes[0] != ApprovalModeAllowOnce {
		t.Fatalf("expected allow_once only, got %+v", items[0].ApprovalModes)
	}

	approval, finalResult, err := manager.DecideApproval(context.Background(), ApprovalDecision{
		CommandID: "approval-1",
		UserID:    "user-1",
		SessionID: "session-1",
		Approved:  true,
		Mode:      ApprovalModeAllowOnce,
	})
	if err != nil {
		t.Fatalf("decide approval: %v", err)
	}
	if approval == nil || finalResult == nil || !finalResult.Success {
		t.Fatalf("expected approval and execution result, got approval=%+v result=%+v", approval, finalResult)
	}
	if transport.lastReq.RequireApproval {
		t.Fatalf("expected approved request to execute without secondary approval")
	}
}

func TestManagerSessionGrantReusesMatchingBinding(t *testing.T) {
	manager := NewManager()
	transport := &managerTestTransport{}
	err := manager.RegisterNode(context.Background(), NodeDescriptor{
		ID: "node-1",
		Capabilities: []NodeCapability{
			{Name: "window.focus"},
		},
	}, transport)
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	req := NodeCommandRequest{
		ID:              "focus-1",
		ActionID:        "action-focus",
		SessionID:       "session-1",
		UserID:          "user-1",
		Capability:      "window.focus",
		Arguments:       map[string]interface{}{"window_handle": "12345", "window_title": "Notepad", "process_name": "notepad.exe"},
		RequireApproval: true,
	}
	result, err := manager.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute with approval: %v", err)
	}
	if pending, _ := result.Data["pending_approval"].(bool); !pending {
		t.Fatalf("expected pending approval response, got %+v", result)
	}

	_, finalResult, err := manager.DecideApproval(context.Background(), ApprovalDecision{
		CommandID: "focus-1",
		SessionID: "session-1",
		UserID:    "user-1",
		Approved:  true,
		Mode:      ApprovalModeAllowSession,
	})
	if err != nil {
		t.Fatalf("decide approval: %v", err)
	}
	if finalResult == nil || !finalResult.Success {
		t.Fatalf("expected approved result, got %+v", finalResult)
	}

	req.ID = "focus-2"
	result, err = manager.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute with session grant: %v", err)
	}
	if pending, _ := result.Data["pending_approval"].(bool); pending {
		t.Fatalf("expected session grant to bypass approval, got %+v", result)
	}
	if transport.lastReq.ApprovalMode != ApprovalModeAllowSession {
		t.Fatalf("expected request to carry allow_session mode, got %+v", transport.lastReq)
	}
}

func TestManagerRejectsExecShellWrapper(t *testing.T) {
	manager := NewManager()
	transport := &managerTestTransport{}
	err := manager.RegisterNode(context.Background(), NodeDescriptor{
		ID: "node-1",
		Capabilities: []NodeCapability{
			{Name: "system.exec"},
		},
	}, transport)
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	_, err = manager.Execute(context.Background(), NodeCommandRequest{
		ID:              "exec-1",
		SessionID:       "session-1",
		UserID:          "user-1",
		Capability:      "system.exec",
		Arguments:       map[string]interface{}{"command": "cmd.exe", "args": []interface{}{"/c", "dir"}},
		RequireApproval: true,
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "blocked") {
		t.Fatalf("expected blocked wrapper error, got %v", err)
	}
}

func TestManagerRejectsExecPathOverride(t *testing.T) {
	manager := NewManager()
	transport := &managerTestTransport{}
	err := manager.RegisterNode(context.Background(), NodeDescriptor{
		ID: "node-1",
		Capabilities: []NodeCapability{
			{Name: "system.exec"},
		},
	}, transport)
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	_, err = manager.Execute(context.Background(), NodeCommandRequest{
		ID:              "exec-2",
		SessionID:       "session-1",
		UserID:          "user-1",
		Capability:      "system.exec",
		Arguments:       map[string]interface{}{"command": "git", "args": []interface{}{"status"}, "env": map[string]interface{}{"PATH": "C:\\bad"}},
		RequireApproval: true,
	})
	if err == nil || !strings.Contains(err.Error(), "PATH") {
		t.Fatalf("expected PATH override error, got %v", err)
	}
}

func TestManagerEmitsLifecycleEvents(t *testing.T) {
	manager := NewManager()
	transport := &managerTestTransport{}
	err := manager.RegisterNode(context.Background(), NodeDescriptor{
		ID: "node-1",
		Capabilities: []NodeCapability{
			{Name: "fs.read"},
		},
	}, transport)
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	baseTime := time.Unix(1700000000, 0)
	manager.now = func() time.Time { return baseTime }
	events := make([]ActionEvent, 0, 3)
	manager.SetEventObserver(func(event ActionEvent) {
		events = append(events, event)
	})

	_, err = manager.Execute(context.Background(), NodeCommandRequest{
		ID:              "read-1",
		ActionID:        "action-read",
		SessionID:       "session-1",
		UserID:          "user-1",
		Capability:      "fs.read",
		Arguments:       map[string]interface{}{"path": "README.md"},
		RequireApproval: true,
	})
	if err != nil {
		t.Fatalf("execute fs.read: %v", err)
	}
	if len(events) < 3 {
		t.Fatalf("expected at least 3 lifecycle events, got %+v", events)
	}
	if events[0].Type != "action.requested" || events[1].Type != "action.started" || events[2].Type != "action.completed" {
		t.Fatalf("unexpected event sequence: %+v", events)
	}
}

func TestManagerWSLExecAllowlistBypassesApproval(t *testing.T) {
	manager := NewManager()
	transport := &managerTestTransport{}
	err := manager.RegisterNode(context.Background(), NodeDescriptor{
		ID: "node-wsl",
		Metadata: map[string]string{
			"kind":                            "wsl",
			"wsl_distro":                      "Ubuntu-22.04",
			"approval_allow_command_prefixes": `["git status"]`,
		},
		Capabilities: []NodeCapability{
			{Name: "wsl.exec"},
		},
	}, transport)
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	result, err := manager.Execute(context.Background(), NodeCommandRequest{
		ID:              "wsl-exec-1",
		Capability:      "wsl.exec",
		Arguments:       map[string]interface{}{"command": "git", "args": []interface{}{"status", "-sb"}},
		RequireApproval: true,
	})
	if err != nil {
		t.Fatalf("execute WSL command: %v", err)
	}
	if pending, _ := result.Data["pending_approval"].(bool); pending {
		t.Fatalf("expected allowlisted WSL command to bypass approval, got %+v", result)
	}
	if transport.lastReq.Capability != "wsl.exec" {
		t.Fatalf("expected request to hit transport, got %+v", transport.lastReq)
	}
}

func TestManagerWSLWriteRequiresApprovalWhenPathIsNotAllowlisted(t *testing.T) {
	manager := NewManager()
	transport := &managerTestTransport{}
	err := manager.RegisterNode(context.Background(), NodeDescriptor{
		ID: "node-wsl",
		Metadata: map[string]string{
			"kind":                               "wsl",
			"wsl_distro":                         "Ubuntu-22.04",
			"approval_allow_write_path_prefixes": `["/workspace/safe"]`,
		},
		Capabilities: []NodeCapability{
			{Name: "wsl.fs.write"},
		},
	}, transport)
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	result, err := manager.Execute(context.Background(), NodeCommandRequest{
		ID:              "wsl-write-1",
		Capability:      "wsl.fs.write",
		Arguments:       map[string]interface{}{"path": "/workspace/other/file.txt", "content": "hello"},
		RequireApproval: true,
	})
	if err != nil {
		t.Fatalf("execute WSL write: %v", err)
	}
	if pending, _ := result.Data["pending_approval"].(bool); !pending {
		t.Fatalf("expected pending approval response, got %+v", result)
	}
	summary, _ := result.Data["summary"].(string)
	if !strings.Contains(summary, "Ubuntu-22.04") || !strings.Contains(summary, "/workspace/other/file.txt") {
		t.Fatalf("unexpected approval summary: %q", summary)
	}
	if transport.lastReq.Capability != "" {
		t.Fatalf("did not expect transport request before approval, got %+v", transport.lastReq)
	}
}
