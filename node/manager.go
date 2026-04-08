package node

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type connectedNode struct {
	desc      NodeDescriptor
	transport Transport
}

type Manager struct {
	mu        sync.RWMutex
	nodes     map[string]*connectedNode
	approvals map[string]ApprovalRequest
}

func NewManager() *Manager {
	return &Manager{
		nodes:     make(map[string]*connectedNode),
		approvals: make(map[string]ApprovalRequest),
	}
}

func (m *Manager) RegisterNode(ctx context.Context, desc NodeDescriptor, transport Transport) error {
	if m == nil {
		return errors.New("node manager is nil")
	}
	if desc.ID == "" {
		return errors.New("node id is required")
	}
	now := time.Now().Unix()
	desc.ConnectedAt = now
	desc.LastSeenAt = now

	m.mu.Lock()
	defer m.mu.Unlock()
	if current, ok := m.nodes[desc.ID]; ok && current.transport != nil {
		_ = current.transport.Close()
	}
	m.nodes[desc.ID] = &connectedNode{
		desc:      desc,
		transport: transport,
	}
	return nil
}

func (m *Manager) RemoveNode(nodeID string) {
	if m == nil || nodeID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.nodes, nodeID)
}

func (m *Manager) Heartbeat(nodeID string) {
	if m == nil || nodeID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if current, ok := m.nodes[nodeID]; ok {
		current.desc.LastSeenAt = time.Now().Unix()
	}
}

func (m *Manager) ListNodes(ctx context.Context) []NodeDescriptor {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make([]NodeDescriptor, 0, len(m.nodes))
	for _, current := range m.nodes {
		items = append(items, current.desc)
	}
	return items
}

func (m *Manager) Execute(ctx context.Context, req NodeCommandRequest) (*NodeCommandResult, error) {
	if m == nil {
		return nil, errors.New("node manager is nil")
	}
	if req.Capability == "" {
		return nil, errors.New("node capability is required")
	}
	if req.ID == "" {
		req.ID = uuid.NewString()
	}

	target, err := m.pickNode(req)
	if err != nil {
		return nil, err
	}
	req.NodeID = target.desc.ID
	if req.RequireApproval {
		approval := ApprovalRequest{
			ID:         req.ID,
			SessionID:  req.SessionID,
			NodeID:     req.NodeID,
			Capability: req.Capability,
			Arguments:  cloneArguments(req.Arguments),
			Summary:    approvalSummary(req),
			CreatedAt:  time.Now().Unix(),
		}
		m.mu.Lock()
		m.approvals[approval.ID] = approval
		m.mu.Unlock()
		return &NodeCommandResult{
			ID:          req.ID,
			NodeID:      req.NodeID,
			Capability:  req.Capability,
			Success:     false,
			Output:      "pending approval",
			StartedAt:   time.Now().Unix(),
			CompletedAt: time.Now().Unix(),
			Data: map[string]interface{}{
				"pending_approval": true,
				"approval_id":      approval.ID,
				"summary":          approval.Summary,
				"arguments":        approval.Arguments,
			},
		}, nil
	}
	return target.transport.Request(ctx, req)
}

func (m *Manager) ListApprovals(ctx context.Context) []ApprovalRequest {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make([]ApprovalRequest, 0, len(m.approvals))
	for _, approval := range m.approvals {
		items = append(items, approval)
	}
	return items
}

func (m *Manager) DecideApproval(ctx context.Context, decision ApprovalDecision) (*ApprovalRequest, *NodeCommandResult, error) {
	if m == nil {
		return nil, nil, errors.New("node manager is nil")
	}
	if decision.CommandID == "" {
		return nil, nil, errors.New("approval command id is required")
	}

	m.mu.Lock()
	approval, ok := m.approvals[decision.CommandID]
	if ok {
		delete(m.approvals, decision.CommandID)
	}
	m.mu.Unlock()
	if !ok {
		return nil, nil, errors.New("approval request not found")
	}

	if !decision.Approved {
		return &approval, nil, nil
	}

	target, err := m.pickNode(NodeCommandRequest{
		NodeID:     approval.NodeID,
		Capability: approval.Capability,
	})
	if err != nil {
		return &approval, nil, err
	}

	req := NodeCommandRequest{
		ID:              approval.ID,
		NodeID:          approval.NodeID,
		SessionID:       approval.SessionID,
		Capability:      approval.Capability,
		Arguments:       cloneArguments(approval.Arguments),
		RequireApproval: false,
	}
	result, err := target.transport.Request(ctx, req)
	return &approval, result, err
}

func (m *Manager) pickNode(req NodeCommandRequest) (*connectedNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if req.NodeID != "" {
		target, ok := m.nodes[req.NodeID]
		if !ok {
			return nil, errors.New("target node not found")
		}
		return target, nil
	}

	for _, current := range m.nodes {
		if supports(current.desc.Capabilities, req.Capability) {
			return current, nil
		}
	}
	return nil, errors.New("no connected node supports this capability")
}

func supports(capabilities []NodeCapability, capability string) bool {
	for _, item := range capabilities {
		if item.Name == capability {
			return true
		}
	}
	return false
}

func cloneArguments(arguments map[string]interface{}) map[string]interface{} {
	if len(arguments) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(arguments))
	for key, value := range arguments {
		cloned[key] = value
	}
	return cloned
}

func approvalSummary(req NodeCommandRequest) string {
	target := approvalTargetSummary(req.Arguments)
	switch req.Capability {
	case "input.keyboard.type":
		text, _ := req.Arguments["text"].(string)
		if target != "" {
			return "Type text on the paired PC into " + target + ": " + truncateText(text, 60)
		}
		return "Type text on the paired PC: " + truncateText(text, 60)
	case "input.keyboard.key":
		key, _ := req.Arguments["key"].(string)
		if target != "" {
			return "Press a key on the paired PC for " + target + ": " + key
		}
		return "Press a key on the paired PC: " + key
	case "input.keyboard.hotkey":
		parts := approvalKeyParts(req.Arguments["keys"])
		if target != "" {
			return "Trigger a hotkey on the paired PC for " + target + ": " + strings.Join(parts, " + ")
		}
		return "Trigger a hotkey on the paired PC: " + strings.Join(parts, " + ")
	case "input.mouse.click", "input.mouse.double_click", "input.mouse.right_click":
		if target != "" {
			return "Click on the paired PC: " + target
		}
		return "Click on the paired PC screen"
	case "input.mouse.drag":
		if target != "" {
			return "Drag on the paired PC from or through " + target
		}
		return "Drag the mouse on the paired PC screen"
	default:
		if target != "" {
			return "Approve PC action: " + req.Capability + " -> " + target
		}
		return "Approve PC action: " + req.Capability
	}
}

func approvalKeyParts(raw interface{}) []string {
	switch values := raw.(type) {
	case []interface{}:
		parts := make([]string, 0, len(values))
		for _, item := range values {
			if value, ok := item.(string); ok && value != "" {
				parts = append(parts, value)
			}
		}
		return parts
	case []string:
		parts := make([]string, 0, len(values))
		for _, value := range values {
			if value != "" {
				parts = append(parts, value)
			}
		}
		return parts
	default:
		return nil
	}
}

func approvalTargetSummary(arguments map[string]interface{}) string {
	if len(arguments) == 0 {
		return ""
	}
	parts := make([]string, 0, 2)
	if raw, ok := arguments["element"].(map[string]interface{}); ok {
		if element := describeElementLocator(raw); element != "" {
			parts = append(parts, element)
		}
	}
	if window := describeWindowLocator(arguments); window != "" {
		parts = append(parts, window)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " in ")
}

func describeElementLocator(locator map[string]interface{}) string {
	if len(locator) == 0 {
		return ""
	}
	candidates := []string{
		truncateText(stringValue(locator["name"]), 40),
		truncateText(stringValue(locator["automation_id"]), 40),
		truncateText(stringValue(locator["path"]), 40),
	}
	role := truncateText(stringValue(locator["role"]), 24)
	className := truncateText(stringValue(locator["class_name"]), 24)
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if role != "" {
			return candidate + " (" + role + ")"
		}
		if className != "" {
			return candidate + " (" + className + ")"
		}
		return candidate
	}
	if role != "" {
		return role
	}
	if className != "" {
		return className
	}
	return ""
}

func describeWindowLocator(locator map[string]interface{}) string {
	if len(locator) == 0 {
		return ""
	}
	for _, key := range []string{"window_title", "title", "process_name", "window_handle", "handle"} {
		if value := truncateText(stringValue(locator[key]), 40); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(raw interface{}) string {
	value, _ := raw.(string)
	return strings.TrimSpace(value)
}

func truncateText(text string, maxLen int) string {
	if maxLen <= 0 || len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
