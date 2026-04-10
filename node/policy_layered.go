package node

import (
	"errors"
	"strings"
	"sync"

	"github.com/LittleSongxx/TinyClaw/authz"
)

type CommandPolicyRule struct {
	Capabilities []string `json:"capabilities,omitempty"`
	Allow        []string `json:"allow,omitempty"`
	Deny         []string `json:"deny,omitempty"`
	BypassScopes []string `json:"bypass_scopes,omitempty"`
}

type LayeredCommandPolicy struct {
	WorkspaceID  string   `json:"workspace_id"`
	DeviceID     string   `json:"device_id,omitempty"`
	Allow        []string `json:"allow,omitempty"`
	Deny         []string `json:"deny,omitempty"`
	BypassRoles  []string `json:"bypass_roles,omitempty"`
	BypassScopes []string `json:"bypass_scopes,omitempty"`
}

var commandPolicyStore = struct {
	sync.RWMutex
	workspace map[string]LayeredCommandPolicy
	device    map[string]LayeredCommandPolicy
}{
	workspace: make(map[string]LayeredCommandPolicy),
	device:    make(map[string]LayeredCommandPolicy),
}

func SetWorkspaceCommandPolicy(policy LayeredCommandPolicy) {
	policy.WorkspaceID = authz.NormalizeWorkspaceID(policy.WorkspaceID)
	commandPolicyStore.Lock()
	defer commandPolicyStore.Unlock()
	commandPolicyStore.workspace[policy.WorkspaceID] = normalizeLayeredPolicy(policy)
}

func SetDeviceCommandPolicy(deviceID string, policy LayeredCommandPolicy) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}
	policy.DeviceID = deviceID
	policy.WorkspaceID = authz.NormalizeWorkspaceID(policy.WorkspaceID)
	commandPolicyStore.Lock()
	defer commandPolicyStore.Unlock()
	commandPolicyStore.device[deviceID] = normalizeLayeredPolicy(policy)
}

func WorkspaceCommandPolicy(workspaceID string) LayeredCommandPolicy {
	workspaceID = authz.NormalizeWorkspaceID(workspaceID)
	commandPolicyStore.RLock()
	defer commandPolicyStore.RUnlock()
	return commandPolicyStore.workspace[workspaceID]
}

func DeviceCommandPolicy(deviceID string) LayeredCommandPolicy {
	commandPolicyStore.RLock()
	defer commandPolicyStore.RUnlock()
	return commandPolicyStore.device[strings.TrimSpace(deviceID)]
}

func EvaluateCommandPolicy(desc NodeDescriptor, req NodeCommandRequest, profile CommandProfile) error {
	if req.WorkspaceID == "" {
		req.WorkspaceID = desc.WorkspaceID
	}
	if !sameWorkspace(req.WorkspaceID, desc.WorkspaceID) {
		return errors.New("node command workspace mismatch")
	}
	if !supports(desc.Capabilities, req.Capability) {
		return errors.New("node did not declare requested capability")
	}
	if baselineDenied(req.Capability) {
		return errors.New("capability is blocked by compiled safety baseline")
	}
	if platformDefaultDenied(desc, req, profile) {
		return errors.New("capability is blocked by platform defaults")
	}

	workspacePolicy := WorkspaceCommandPolicy(req.WorkspaceID)
	if containsPolicyValue(workspacePolicy.Deny, req.Capability) {
		return errors.New("capability denied by workspace policy")
	}
	if len(workspacePolicy.Allow) > 0 && !containsPolicyValue(workspacePolicy.Allow, req.Capability) {
		return errors.New("capability is not allowed by workspace policy")
	}

	devicePolicy := DeviceCommandPolicy(firstNonEmpty(desc.DeviceID, desc.ID))
	if containsPolicyValue(devicePolicy.Deny, req.Capability) {
		return errors.New("capability denied by device policy")
	}
	if len(devicePolicy.Allow) > 0 && !containsPolicyValue(devicePolicy.Allow, req.Capability) {
		return errors.New("capability is not allowed by device policy")
	}

	role := authz.NormalizeRole(authz.Role(req.ActorRole))
	if profile.Mutating && role == authz.RoleViewer {
		return errors.New("viewer cannot execute mutating node commands")
	}
	if profile.Risk == RiskHigh && !(role == authz.RoleOwner || role == authz.RoleAdmin || role == authz.RoleOperator) {
		return errors.New("high risk node command requires owner, admin, or operator role")
	}
	return nil
}

func baselineDenied(capability string) bool {
	switch strings.TrimSpace(capability) {
	case "", "system.shell", "eval", "plugin.dynamic.go":
		return true
	default:
		return false
	}
}

func platformDefaultDenied(desc NodeDescriptor, req NodeCommandRequest, profile CommandProfile) bool {
	platform := normalizePlatformFamily(desc)
	if platform == "unknown" && (profile.Mutating || strings.Contains(req.Capability, ".exec") || strings.Contains(req.Capability, ".write") || strings.HasPrefix(req.Capability, "input.")) {
		return true
	}
	return false
}

func normalizePlatformFamily(desc NodeDescriptor) string {
	value := strings.ToLower(strings.TrimSpace(desc.Platform))
	if value == "" && len(desc.Metadata) > 0 {
		value = strings.ToLower(strings.TrimSpace(desc.Metadata[metadataKindKey]))
	}
	switch value {
	case "windows", "win32":
		return "windows"
	case "wsl":
		return "wsl"
	case "linux":
		return "linux"
	case "darwin", "macos":
		return "macos"
	case "browser":
		return "browser"
	default:
		return "unknown"
	}
}

func hasPolicyBypass(role authz.Role, scopes []string, policies ...LayeredCommandPolicy) bool {
	for _, policy := range policies {
		for _, allowedRole := range policy.BypassRoles {
			if authz.NormalizeRole(authz.Role(allowedRole)) == role {
				return true
			}
		}
		for _, allowedScope := range policy.BypassScopes {
			for _, scope := range scopes {
				if strings.EqualFold(scope, allowedScope) || scope == "*" {
					return true
				}
			}
		}
	}
	return false
}

func normalizeLayeredPolicy(policy LayeredCommandPolicy) LayeredCommandPolicy {
	policy.Allow = normalizePolicyValues(policy.Allow)
	policy.Deny = normalizePolicyValues(policy.Deny)
	policy.BypassRoles = normalizePolicyValues(policy.BypassRoles)
	policy.BypassScopes = normalizePolicyValues(policy.BypassScopes)
	return policy
}

func normalizePolicyValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func containsPolicyValue(values []string, capability string) bool {
	capability = strings.ToLower(strings.TrimSpace(capability))
	for _, value := range values {
		if value == "*" || value == capability {
			return true
		}
		if strings.HasSuffix(value, ".*") && strings.HasPrefix(capability, strings.TrimSuffix(value, "*")) {
			return true
		}
	}
	return false
}
