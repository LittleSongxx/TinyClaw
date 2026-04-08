package node

import (
	"context"
	"testing"
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

	approval, finalResult, err := manager.DecideApproval(context.Background(), ApprovalDecision{
		CommandID: "approval-1",
		Approved:  true,
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
