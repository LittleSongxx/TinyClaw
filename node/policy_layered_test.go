package node

import (
	"strings"
	"testing"

	"github.com/LittleSongxx/TinyClaw/authz"
)

func TestLayeredPolicyDeniesViewerMutation(t *testing.T) {
	desc := NodeDescriptor{
		ID:          "node-1",
		WorkspaceID: "default",
		Platform:    "windows",
		Capabilities: []NodeCapability{
			{Name: "fs.write"},
		},
	}
	req := NodeCommandRequest{
		WorkspaceID:     "default",
		ActorRole:       string(authz.RoleViewer),
		Capability:      "fs.write",
		RequireApproval: true,
	}
	err := EvaluateCommandPolicy(desc, req, commandProfile("fs.write"))
	if err == nil || !strings.Contains(err.Error(), "viewer") {
		t.Fatalf("expected viewer mutation denial, got %v", err)
	}
}

func TestLayeredPolicyUnknownPlatformDeniesHostMutation(t *testing.T) {
	desc := NodeDescriptor{
		ID:          "node-1",
		WorkspaceID: "default",
		Platform:    "unknown",
		Capabilities: []NodeCapability{
			{Name: "system.exec"},
		},
	}
	req := NodeCommandRequest{
		WorkspaceID:     "default",
		ActorRole:       string(authz.RoleOperator),
		Capability:      "system.exec",
		RequireApproval: true,
	}
	err := EvaluateCommandPolicy(desc, req, commandProfile("system.exec"))
	if err == nil || !strings.Contains(err.Error(), "platform") {
		t.Fatalf("expected unknown platform denial, got %v", err)
	}
}

func TestLayeredPolicyWorkspaceDenyOverridesAllow(t *testing.T) {
	SetWorkspaceCommandPolicy(LayeredCommandPolicy{WorkspaceID: "policy-test", Allow: []string{"*"}, Deny: []string{"fs.write"}})
	desc := NodeDescriptor{
		ID:          "node-1",
		WorkspaceID: "policy-test",
		Platform:    "windows",
		Capabilities: []NodeCapability{
			{Name: "fs.write"},
		},
	}
	req := NodeCommandRequest{
		WorkspaceID:     "policy-test",
		ActorRole:       string(authz.RoleOperator),
		Capability:      "fs.write",
		RequireApproval: true,
	}
	err := EvaluateCommandPolicy(desc, req, commandProfile("fs.write"))
	if err == nil || !strings.Contains(err.Error(), "workspace policy") {
		t.Fatalf("expected workspace deny, got %v", err)
	}
}
