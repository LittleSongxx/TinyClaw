package node

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
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
	grants    map[string][]ApprovalGrant
	observer  ActionEventObserver
	now       func() time.Time
}

func NewManager() *Manager {
	return &Manager{
		nodes:     make(map[string]*connectedNode),
		approvals: make(map[string]ApprovalRequest),
		grants:    make(map[string][]ApprovalGrant),
		now:       time.Now,
	}
}

func (m *Manager) SetEventObserver(observer ActionEventObserver) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.observer = observer
}

func (m *Manager) RegisterNode(ctx context.Context, desc NodeDescriptor, transport Transport) error {
	if m == nil {
		return errors.New("node manager is nil")
	}
	if desc.ID == "" {
		return errors.New("node id is required")
	}
	desc.WorkspaceID = authz.NormalizeWorkspaceID(desc.WorkspaceID)
	if desc.Platform == "" {
		desc.Platform = "windows"
	}
	if desc.DeviceID == "" {
		desc.DeviceID = desc.ID
	}
	now := m.timeNow().Unix()
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
		current.desc.LastSeenAt = m.timeNow().Unix()
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
		if !sameWorkspace(authz.WorkspaceIDFromContext(ctx), current.desc.WorkspaceID) {
			continue
		}
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
	if req.ActionID == "" {
		req.ActionID = req.ID
	}
	if principal, ok := authz.PrincipalFromContext(ctx); ok {
		req.WorkspaceID = authz.NormalizeWorkspaceID(firstNonEmpty(req.WorkspaceID, principal.WorkspaceID))
		req.ActorID = firstNonEmpty(req.ActorID, principal.ActorID)
		req.ActorRole = firstNonEmpty(req.ActorRole, string(principal.Role))
		if len(req.ActorScopes) == 0 {
			req.ActorScopes = append([]string(nil), principal.Scopes...)
		}
	} else {
		req.WorkspaceID = authz.NormalizeWorkspaceID(req.WorkspaceID)
	}
	if req.ActorID == "" && req.UserID != "" {
		req.ActorID = req.UserID
	}
	if req.ActorRole == "" {
		req.ActorRole = string(authz.RoleOperator)
	}

	target, err := m.pickNode(req)
	if err != nil {
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   req.ActionID,
			SessionID:  req.SessionID,
			UserID:     req.UserID,
			Capability: req.Capability,
			Detail:     err.Error(),
			CreatedAt:  m.timeNow().Unix(),
		})
		return nil, err
	}
	req.NodeID = target.desc.ID

	now := m.timeNow()
	req, profile, binding, err := normalizeNodeRequest(target.desc, req, now)
	summary := approvalSummary(req, target.desc)
	m.emit(ActionEvent{
		Type:       "action.requested",
		ActionID:   req.ActionID,
		SessionID:  req.SessionID,
		UserID:     req.UserID,
		NodeID:     req.NodeID,
		Capability: req.Capability,
		Summary:    summary,
		CreatedAt:  now.Unix(),
	})
	if err != nil {
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   req.ActionID,
			SessionID:  req.SessionID,
			UserID:     req.UserID,
			NodeID:     req.NodeID,
			Capability: req.Capability,
			Summary:    summary,
			Detail:     err.Error(),
			CreatedAt:  m.timeNow().Unix(),
		})
		return nil, err
	}
	if err := EvaluateCommandPolicy(target.desc, req, profile); err != nil {
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   req.ActionID,
			SessionID:  req.SessionID,
			UserID:     req.UserID,
			NodeID:     req.NodeID,
			Capability: req.Capability,
			Summary:    summary,
			Detail:     err.Error(),
			CreatedAt:  m.timeNow().Unix(),
		})
		return nil, err
	}

	if grant := m.matchGrant(req, binding, now); grant != nil {
		req.ApprovalMode = grant.Mode
		return m.dispatch(ctx, target, req, summary)
	}

	modes := allowedApprovalModes(req, profile, binding)
	if len(modes) == 0 {
		return m.dispatch(ctx, target, req, summary)
	}

	approval := ApprovalRequest{
		ID:            req.ID,
		WorkspaceID:   req.WorkspaceID,
		ActorID:       req.ActorID,
		ActorRole:     req.ActorRole,
		ActionID:      req.ActionID,
		SessionID:     req.SessionID,
		UserID:        req.UserID,
		NodeID:        req.NodeID,
		Capability:    req.Capability,
		Arguments:     cloneArguments(req.Arguments),
		Summary:       summary,
		Binding:       binding,
		ApprovalModes: append([]ApprovalMode(nil), modes...),
		CreatedAt:     now.Unix(),
		ExpiresAt:     now.Add(sessionGrantTTL).Unix(),
	}

	m.mu.Lock()
	m.cleanupLocked(now)
	m.approvals[approval.ID] = approval
	m.mu.Unlock()

	m.emit(ActionEvent{
		Type:       "approval.requested",
		ActionID:   req.ActionID,
		ApprovalID: approval.ID,
		SessionID:  approval.SessionID,
		UserID:     approval.UserID,
		NodeID:     approval.NodeID,
		Capability: approval.Capability,
		Summary:    approval.Summary,
		CreatedAt:  now.Unix(),
		Metadata: map[string]string{
			"approval_modes": joinApprovalModes(approval.ApprovalModes),
		},
	})

	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Success:     false,
		Output:      "pending approval",
		StartedAt:   now.Unix(),
		CompletedAt: now.Unix(),
		Data: map[string]interface{}{
			"pending_approval": true,
			"approval_id":      approval.ID,
			"summary":          approval.Summary,
			"arguments":        approval.Arguments,
			"approval_modes":   approval.ApprovalModes,
			"session_id":       approval.SessionID,
		},
	}, nil
}

func (m *Manager) ListApprovals(ctx context.Context) []ApprovalRequest {
	if m == nil {
		return nil
	}
	now := m.timeNow()
	m.mu.Lock()
	m.cleanupLocked(now)
	items := make([]ApprovalRequest, 0, len(m.approvals))
	workspaceID := authz.WorkspaceIDFromContext(ctx)
	for _, approval := range m.approvals {
		if !sameWorkspace(workspaceID, approval.WorkspaceID) {
			continue
		}
		items = append(items, approval)
	}
	m.mu.Unlock()
	return items
}

func (m *Manager) DecideApproval(ctx context.Context, decision ApprovalDecision) (*ApprovalRequest, *NodeCommandResult, error) {
	if m == nil {
		return nil, nil, errors.New("node manager is nil")
	}
	if decision.CommandID == "" {
		return nil, nil, errors.New("approval command id is required")
	}
	if principal, ok := authz.PrincipalFromContext(ctx); ok {
		decision.WorkspaceID = authz.NormalizeWorkspaceID(firstNonEmpty(decision.WorkspaceID, principal.WorkspaceID))
		decision.ActorID = firstNonEmpty(decision.ActorID, principal.ActorID)
		decision.ActorRole = firstNonEmpty(decision.ActorRole, string(principal.Role))
	} else {
		decision.WorkspaceID = authz.NormalizeWorkspaceID(decision.WorkspaceID)
	}
	if decision.ActorRole == "" {
		decision.ActorRole = string(authz.RoleOperator)
	}

	now := m.timeNow()
	mode := approvalMode(decision)
	if mode == "" {
		mode = ApprovalModeReject
	}

	var approval ApprovalRequest
	m.mu.Lock()
	m.cleanupLocked(now)
	current, ok := m.approvals[decision.CommandID]
	if ok {
		delete(m.approvals, decision.CommandID)
	}
	if ok {
		approval = current
	}
	m.mu.Unlock()
	if !ok {
		m.emit(ActionEvent{
			Type:      "action.denied",
			ActionID:  decision.CommandID,
			SessionID: decision.SessionID,
			UserID:    decision.UserID,
			NodeID:    decision.NodeID,
			Detail:    "approval request not found",
			CreatedAt: now.Unix(),
		})
		return nil, nil, errors.New("approval request not found")
	}
	if !sameWorkspace(decision.WorkspaceID, approval.WorkspaceID) {
		return &approval, nil, errors.New("approval belongs to a different workspace")
	}
	if !canResolveApproval(decision.ActorRole) {
		return &approval, nil, errors.New("owner, admin, or operator role is required to resolve approval")
	}

	if approval.ExpiresAt > 0 && approval.ExpiresAt <= now.Unix() {
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   approval.ActionID,
			ApprovalID: approval.ID,
			SessionID:  approval.SessionID,
			UserID:     decision.UserID,
			NodeID:     approval.NodeID,
			Capability: approval.Capability,
			Summary:    approval.Summary,
			Detail:     "approval request expired",
			CreatedAt:  now.Unix(),
		})
		return nil, nil, errors.New("approval request expired")
	}
	if approval.SessionID != "" && decision.SessionID != "" && approval.SessionID != decision.SessionID {
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   approval.ActionID,
			ApprovalID: approval.ID,
			SessionID:  decision.SessionID,
			UserID:     decision.UserID,
			NodeID:     approval.NodeID,
			Capability: approval.Capability,
			Summary:    approval.Summary,
			Detail:     "approval session mismatch",
			CreatedAt:  now.Unix(),
		})
		return &approval, nil, errors.New("approval session mismatch")
	}
	if approval.UserID != "" && decision.UserID != "" && approval.UserID != decision.UserID {
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   approval.ActionID,
			ApprovalID: approval.ID,
			SessionID:  approval.SessionID,
			UserID:     decision.UserID,
			NodeID:     approval.NodeID,
			Capability: approval.Capability,
			Summary:    approval.Summary,
			Detail:     "approval user mismatch",
			CreatedAt:  now.Unix(),
		})
		return &approval, nil, errors.New("approval user mismatch")
	}

	m.emit(ActionEvent{
		Type:       "approval.resolved",
		ActionID:   approval.ActionID,
		ApprovalID: approval.ID,
		SessionID:  approval.SessionID,
		UserID:     firstNonEmpty(decision.UserID, approval.UserID),
		NodeID:     approval.NodeID,
		Capability: approval.Capability,
		Summary:    approval.Summary,
		Mode:       mode,
		CreatedAt:  now.Unix(),
	})

	if mode == ApprovalModeReject {
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   approval.ActionID,
			ApprovalID: approval.ID,
			SessionID:  approval.SessionID,
			UserID:     firstNonEmpty(decision.UserID, approval.UserID),
			NodeID:     approval.NodeID,
			Capability: approval.Capability,
			Summary:    approval.Summary,
			Detail:     "rejected by user",
			CreatedAt:  now.Unix(),
		})
		return &approval, nil, nil
	}

	if !containsApprovalMode(approval.ApprovalModes, mode) {
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   approval.ActionID,
			ApprovalID: approval.ID,
			SessionID:  approval.SessionID,
			UserID:     firstNonEmpty(decision.UserID, approval.UserID),
			NodeID:     approval.NodeID,
			Capability: approval.Capability,
			Summary:    approval.Summary,
			Detail:     "approval mode not allowed",
			CreatedAt:  now.Unix(),
		})
		return &approval, nil, errors.New("approval mode not allowed for this action")
	}

	if mode == ApprovalModeAllowSession {
		grant := ApprovalGrant{
			ID:          approval.ID,
			WorkspaceID: approval.WorkspaceID,
			SessionID:   approval.SessionID,
			UserID:      approval.UserID,
			NodeID:      approval.NodeID,
			Capability:  approval.Capability,
			Mode:        mode,
			Summary:     approval.Summary,
			Binding: ApprovalBinding{
				WorkspaceID:    approval.Binding.WorkspaceID,
				SessionID:      approval.Binding.SessionID,
				UserID:         approval.Binding.UserID,
				NodeID:         approval.Binding.NodeID,
				Capability:     approval.Binding.Capability,
				ArgsDigest:     approval.Binding.ArgsDigest,
				WindowBinding:  approval.Binding.WindowBinding,
				ElementBinding: approval.Binding.ElementBinding,
				CreatedAt:      now.Unix(),
				ExpiresAt:      now.Add(sessionGrantTTL).Unix(),
			},
			CreatedAt: now.Unix(),
			ExpiresAt: now.Add(sessionGrantTTL).Unix(),
		}
		m.mu.Lock()
		existing := m.grants[grant.SessionID]
		existing = append(existing, grant)
		m.grants[grant.SessionID] = existing
		m.mu.Unlock()
	}

	target, err := m.pickNode(NodeCommandRequest{
		WorkspaceID: approval.WorkspaceID,
		NodeID:      approval.NodeID,
		Capability:  approval.Capability,
	})
	if err != nil {
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   approval.ActionID,
			ApprovalID: approval.ID,
			SessionID:  approval.SessionID,
			UserID:     firstNonEmpty(decision.UserID, approval.UserID),
			NodeID:     approval.NodeID,
			Capability: approval.Capability,
			Summary:    approval.Summary,
			Detail:     err.Error(),
			CreatedAt:  now.Unix(),
		})
		return &approval, nil, err
	}

	req := NodeCommandRequest{
		ID:              approval.ID,
		WorkspaceID:     approval.WorkspaceID,
		ActorID:         approval.ActorID,
		ActorRole:       approval.ActorRole,
		NodeID:          approval.NodeID,
		SessionID:       approval.SessionID,
		UserID:          approval.UserID,
		Capability:      approval.Capability,
		Arguments:       cloneArguments(approval.Arguments),
		RequireApproval: false,
		ApprovalMode:    mode,
		ActionID:        firstNonEmpty(approval.ActionID, approval.ID),
		BindingHint: map[string]interface{}{
			"args_digest":     approval.Binding.ArgsDigest,
			"window_binding":  approval.Binding.WindowBinding,
			"element_binding": approval.Binding.ElementBinding,
		},
	}
	result, err := m.dispatch(ctx, target, req, approval.Summary)
	return &approval, result, err
}

func (m *Manager) dispatch(ctx context.Context, target *connectedNode, req NodeCommandRequest, summary string) (*NodeCommandResult, error) {
	m.emit(ActionEvent{
		Type:       "action.started",
		ActionID:   req.ActionID,
		SessionID:  req.SessionID,
		UserID:     req.UserID,
		NodeID:     req.NodeID,
		Capability: req.Capability,
		Summary:    summary,
		Mode:       req.ApprovalMode,
		CreatedAt:  m.timeNow().Unix(),
	})

	result, err := target.transport.Request(ctx, req)
	now := m.timeNow().Unix()
	if err != nil {
		detail := err.Error()
		if result != nil && result.Error != "" {
			detail = result.Error
		}
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   req.ActionID,
			SessionID:  req.SessionID,
			UserID:     req.UserID,
			NodeID:     req.NodeID,
			Capability: req.Capability,
			Summary:    summary,
			Detail:     detail,
			Mode:       req.ApprovalMode,
			CreatedAt:  now,
		})
		return result, err
	}
	if result == nil {
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   req.ActionID,
			SessionID:  req.SessionID,
			UserID:     req.UserID,
			NodeID:     req.NodeID,
			Capability: req.Capability,
			Summary:    summary,
			Detail:     "empty node response",
			Mode:       req.ApprovalMode,
			CreatedAt:  now,
		})
		return nil, errors.New("empty node response")
	}
	if !result.Success || result.Error != "" {
		detail := firstNonEmpty(result.Error, result.Output)
		m.emit(ActionEvent{
			Type:       "action.denied",
			ActionID:   req.ActionID,
			SessionID:  req.SessionID,
			UserID:     req.UserID,
			NodeID:     req.NodeID,
			Capability: req.Capability,
			Summary:    summary,
			Detail:     detail,
			Mode:       req.ApprovalMode,
			CreatedAt:  now,
		})
		return result, nil
	}
	m.emit(ActionEvent{
		Type:       "action.completed",
		ActionID:   req.ActionID,
		SessionID:  req.SessionID,
		UserID:     req.UserID,
		NodeID:     req.NodeID,
		Capability: req.Capability,
		Summary:    summary,
		Detail:     firstNonEmpty(result.Output, "ok"),
		Mode:       req.ApprovalMode,
		Success:    true,
		CreatedAt:  now,
	})
	return result, nil
}

func (m *Manager) matchGrant(req NodeCommandRequest, binding ApprovalBinding, now time.Time) *ApprovalGrant {
	if req.SessionID == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	items := m.grants[req.SessionID]
	for index := range items {
		item := &items[index]
		if !sameWorkspace(item.WorkspaceID, req.WorkspaceID) || item.SessionID != req.SessionID || item.UserID != req.UserID || item.NodeID != req.NodeID || item.Capability != req.Capability {
			continue
		}
		if !bindingMatches(item.Binding, binding) {
			continue
		}
		item.ExpiresAt = now.Add(sessionGrantTTL).Unix()
		item.Binding.ExpiresAt = item.ExpiresAt
		m.grants[req.SessionID] = items
		copyItem := *item
		return &copyItem
	}
	return nil
}

func (m *Manager) cleanupLocked(now time.Time) {
	nowUnix := now.Unix()
	for id, approval := range m.approvals {
		if approval.ExpiresAt > 0 && approval.ExpiresAt <= nowUnix {
			delete(m.approvals, id)
		}
	}
	for sessionID, items := range m.grants {
		filtered := items[:0]
		for _, item := range items {
			if item.ExpiresAt > 0 && item.ExpiresAt <= nowUnix {
				continue
			}
			filtered = append(filtered, item)
		}
		if len(filtered) == 0 {
			delete(m.grants, sessionID)
			continue
		}
		m.grants[sessionID] = filtered
	}
}

func (m *Manager) emit(event ActionEvent) {
	if m == nil {
		return
	}
	m.mu.RLock()
	observer := m.observer
	m.mu.RUnlock()
	if observer != nil {
		observer(event)
	}
}

func (m *Manager) timeNow() time.Time {
	if m != nil && m.now != nil {
		return m.now()
	}
	return time.Now()
}

func (m *Manager) pickNode(req NodeCommandRequest) (*connectedNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if req.NodeID != "" {
		target, ok := m.nodes[req.NodeID]
		if !ok {
			return nil, errors.New("target node not found")
		}
		if !sameWorkspace(req.WorkspaceID, target.desc.WorkspaceID) {
			return nil, errors.New("target node belongs to a different workspace")
		}
		if !supports(target.desc.Capabilities, req.Capability) {
			return nil, errors.New("target node does not declare this capability")
		}
		return target, nil
	}

	candidates := make([]*connectedNode, 0, len(m.nodes))
	for _, current := range m.nodes {
		if sameWorkspace(req.WorkspaceID, current.desc.WorkspaceID) && supports(current.desc.Capabilities, req.Capability) {
			candidates = append(candidates, current)
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("no connected node supports this capability")
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].desc.LastSeenAt != candidates[j].desc.LastSeenAt {
			return candidates[i].desc.LastSeenAt > candidates[j].desc.LastSeenAt
		}
		return candidates[i].desc.ID < candidates[j].desc.ID
	})
	return candidates[0], nil
}

func sameWorkspace(left, right string) bool {
	return authz.NormalizeWorkspaceID(left) == authz.NormalizeWorkspaceID(right)
}

func canResolveApproval(role string) bool {
	switch authz.NormalizeRole(authz.Role(role)) {
	case authz.RoleOwner, authz.RoleAdmin, authz.RoleOperator:
		return true
	default:
		return false
	}
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

func containsApprovalMode(items []ApprovalMode, target ApprovalMode) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func bindingMatches(left, right ApprovalBinding) bool {
	return sameWorkspace(left.WorkspaceID, right.WorkspaceID) &&
		left.SessionID == right.SessionID &&
		left.UserID == right.UserID &&
		left.NodeID == right.NodeID &&
		left.Capability == right.Capability &&
		left.ArgsDigest == right.ArgsDigest &&
		left.WindowBinding == right.WindowBinding &&
		left.ElementBinding == right.ElementBinding
}

func joinApprovalModes(items []ApprovalMode) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, string(item))
	}
	return strings.Join(parts, ",")
}

func approvalSummary(req NodeCommandRequest, desc NodeDescriptor) string {
	target := approvalTargetSummary(req.Arguments)
	switch req.Capability {
	case "system.exec":
		command := previewSensitiveText(buildApprovalCommandLine(req.Arguments), resultRedactionHash, 32)
		if dir := stringArg(req.Arguments, "dir"); dir != "" {
			return "Execute command on the paired PC in " + truncateText(dir, 40) + ": " + command
		}
		return "Execute command on the paired PC: " + command
	case "fs.write":
		targetPath := truncateText(stringArg(req.Arguments, "path"), 60)
		return "Write file on the paired PC: " + targetPath
	case "input.keyboard.type":
		text := previewSensitiveText(stringArg(req.Arguments, "text"), resultRedactionHash, 24)
		if target != "" {
			return "Type text on the paired PC into " + target + ": " + text
		}
		return "Type text on the paired PC: " + text
	case "input.keyboard.key":
		key, _ := req.Arguments["key"].(string)
		if target != "" {
			return "Press a key on the paired PC for " + target + ": " + key
		}
		return "Press a key on the paired PC: " + key
	case "input.keyboard.hotkey":
		parts := approvalKeyParts(req.Arguments["keys"])
		keys := previewSensitiveText(strings.Join(parts, " + "), resultRedactionTruncate, 40)
		if target != "" {
			return "Trigger a hotkey on the paired PC for " + target + ": " + keys
		}
		return "Trigger a hotkey on the paired PC: " + keys
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
	case "window.focus":
		return "Focus a desktop window on the paired PC: " + firstNonEmpty(target, "specified window")
	case "ui.focus":
		return "Focus a desktop UI element on the paired PC: " + firstNonEmpty(target, "specified element")
	case "wsl.exec":
		command := previewSensitiveText(buildApprovalCommandLine(req.Arguments), resultRedactionHash, 32)
		distro := wslDistroName(desc)
		if distro != "" {
			return "Execute WSL command on " + distro + ": " + command
		}
		return "Execute WSL command: " + command
	case "wsl.fs.write":
		pathText := normalizeApprovalPath(stringArg(req.Arguments, "path"))
		distro := wslDistroName(desc)
		if distro != "" {
			return "Write file in " + distro + ": " + truncateText(pathText, 60)
		}
		return "Write file in WSL: " + truncateText(pathText, 60)
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
	role := truncateText(firstNonEmptyString(locator, "role", "control_type"), 24)
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

func previewSensitiveText(text string, strategy string, maxLen int) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	switch strategy {
	case resultRedactionHash:
		preview := truncateText(trimmed, maxLen)
		return preview + " [sha1:" + shortHashString(trimmed) + "]"
	case resultRedactionTruncate:
		return truncateText(trimmed, maxLen)
	default:
		return trimmed
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
