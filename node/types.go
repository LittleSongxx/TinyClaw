package node

import "context"

type NodeCapability struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type NodeDescriptor struct {
	ID           string            `json:"id"`
	WorkspaceID  string            `json:"workspace_id"`
	DeviceID     string            `json:"device_id,omitempty"`
	Name         string            `json:"name"`
	Platform     string            `json:"platform"`
	Hostname     string            `json:"hostname"`
	Version      string            `json:"version"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Capabilities []NodeCapability  `json:"capabilities,omitempty"`
	ConnectedAt  int64             `json:"connected_at"`
	LastSeenAt   int64             `json:"last_seen_at"`
}

type NodeCommandRequest struct {
	ID              string                 `json:"id"`
	WorkspaceID     string                 `json:"workspace_id,omitempty"`
	ActorID         string                 `json:"actor_id,omitempty"`
	ActorRole       string                 `json:"actor_role,omitempty"`
	ActorScopes     []string               `json:"actor_scopes,omitempty"`
	NodeID          string                 `json:"node_id,omitempty"`
	SessionID       string                 `json:"session_id,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	Capability      string                 `json:"capability"`
	Arguments       map[string]interface{} `json:"arguments,omitempty"`
	TimeoutSec      int                    `json:"timeout_sec,omitempty"`
	RequireApproval bool                   `json:"require_approval,omitempty"`
	ApprovalMode    ApprovalMode           `json:"approval_mode,omitempty"`
	ActionID        string                 `json:"action_id,omitempty"`
	BindingHint     map[string]interface{} `json:"binding_hint,omitempty"`
}

type ApprovalMode string

const (
	ApprovalModeAllowOnce    ApprovalMode = "allow_once"
	ApprovalModeAllowSession ApprovalMode = "allow_session"
	ApprovalModeReject       ApprovalMode = "reject"
)

type ApprovalBinding struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	NodeID         string `json:"node_id,omitempty"`
	Capability     string `json:"capability,omitempty"`
	ArgsDigest     string `json:"args_digest,omitempty"`
	WindowBinding  string `json:"window_binding,omitempty"`
	ElementBinding string `json:"element_binding,omitempty"`
	CreatedAt      int64  `json:"created_at"`
	ExpiresAt      int64  `json:"expires_at"`
}

type ApprovalGrant struct {
	ID         string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	SessionID  string          `json:"session_id,omitempty"`
	UserID     string          `json:"user_id,omitempty"`
	NodeID     string          `json:"node_id"`
	Capability string          `json:"capability"`
	Mode       ApprovalMode    `json:"mode"`
	Summary    string          `json:"summary,omitempty"`
	Binding    ApprovalBinding `json:"binding"`
	CreatedAt  int64           `json:"created_at"`
	ExpiresAt  int64           `json:"expires_at"`
}

type ApprovalRequest struct {
	ID            string                 `json:"id"`
	WorkspaceID   string                 `json:"workspace_id"`
	ActorID       string                 `json:"actor_id,omitempty"`
	ActorRole     string                 `json:"actor_role,omitempty"`
	ActionID      string                 `json:"action_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	NodeID        string                 `json:"node_id"`
	Capability    string                 `json:"capability"`
	Arguments     map[string]interface{} `json:"arguments,omitempty"`
	Summary       string                 `json:"summary,omitempty"`
	Binding       ApprovalBinding        `json:"binding"`
	ApprovalModes []ApprovalMode         `json:"approval_modes,omitempty"`
	CreatedAt     int64                  `json:"created_at"`
	ExpiresAt     int64                  `json:"expires_at"`
}

type NodeCommandResult struct {
	ID          string                 `json:"id"`
	NodeID      string                 `json:"node_id"`
	Capability  string                 `json:"capability"`
	Success     bool                   `json:"success"`
	Output      string                 `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	StartedAt   int64                  `json:"started_at"`
	CompletedAt int64                  `json:"completed_at"`
}

type ApprovalDecision struct {
	ID        string       `json:"id"`
	WorkspaceID string       `json:"workspace_id"`
	ActorID   string       `json:"actor_id,omitempty"`
	ActorRole string       `json:"actor_role,omitempty"`
	CommandID string       `json:"command_id"`
	SessionID string       `json:"session_id,omitempty"`
	UserID    string       `json:"user_id,omitempty"`
	NodeID    string       `json:"node_id"`
	Approved  bool         `json:"approved"`
	Mode      ApprovalMode `json:"mode,omitempty"`
	Reason    string       `json:"reason,omitempty"`
	CreatedAt int64        `json:"created_at"`
}

type ActionEvent struct {
	Type       string            `json:"type"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
	ActorID    string            `json:"actor_id,omitempty"`
	ActionID   string            `json:"action_id,omitempty"`
	ApprovalID string            `json:"approval_id,omitempty"`
	SessionID  string            `json:"session_id,omitempty"`
	UserID     string            `json:"user_id,omitempty"`
	NodeID     string            `json:"node_id,omitempty"`
	Capability string            `json:"capability,omitempty"`
	Summary    string            `json:"summary,omitempty"`
	Detail     string            `json:"detail,omitempty"`
	Mode       ApprovalMode      `json:"mode,omitempty"`
	Success    bool              `json:"success,omitempty"`
	CreatedAt  int64             `json:"created_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type ActionEventObserver func(ActionEvent)

type Transport interface {
	Request(ctx context.Context, req NodeCommandRequest) (*NodeCommandResult, error)
	Close() error
}

type Broker interface {
	RegisterNode(ctx context.Context, desc NodeDescriptor, transport Transport) error
	RemoveNode(nodeID string)
	Heartbeat(nodeID string)
	ListNodes(ctx context.Context) []NodeDescriptor
	Execute(ctx context.Context, req NodeCommandRequest) (*NodeCommandResult, error)
	ListApprovals(ctx context.Context) []ApprovalRequest
	DecideApproval(ctx context.Context, decision ApprovalDecision) (*ApprovalRequest, *NodeCommandResult, error)
}

type Driver interface {
	Capabilities() []NodeCapability
	Execute(ctx context.Context, req NodeCommandRequest) (*NodeCommandResult, error)
}
