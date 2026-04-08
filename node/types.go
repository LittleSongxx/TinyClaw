package node

import "context"

type NodeCapability struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type NodeDescriptor struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Platform     string           `json:"platform"`
	Hostname     string           `json:"hostname"`
	Version      string           `json:"version"`
	Capabilities []NodeCapability `json:"capabilities,omitempty"`
	ConnectedAt  int64            `json:"connected_at"`
	LastSeenAt   int64            `json:"last_seen_at"`
}

type NodeCommandRequest struct {
	ID              string                 `json:"id"`
	NodeID          string                 `json:"node_id,omitempty"`
	SessionID       string                 `json:"session_id,omitempty"`
	Capability      string                 `json:"capability"`
	Arguments       map[string]interface{} `json:"arguments,omitempty"`
	TimeoutSec      int                    `json:"timeout_sec,omitempty"`
	RequireApproval bool                   `json:"require_approval,omitempty"`
}

type ApprovalRequest struct {
	ID         string                 `json:"id"`
	SessionID  string                 `json:"session_id,omitempty"`
	NodeID     string                 `json:"node_id"`
	Capability string                 `json:"capability"`
	Arguments  map[string]interface{} `json:"arguments,omitempty"`
	Summary    string                 `json:"summary,omitempty"`
	CreatedAt  int64                  `json:"created_at"`
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
	ID        string `json:"id"`
	CommandID string `json:"command_id"`
	SessionID string `json:"session_id,omitempty"`
	NodeID    string `json:"node_id"`
	Approved  bool   `json:"approved"`
	Reason    string `json:"reason,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

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
