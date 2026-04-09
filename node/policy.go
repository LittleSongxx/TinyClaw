package node

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

const (
	execSanitizerNone       = ""
	execSanitizerStructured = "structured_argv"

	resultRedactionNone     = ""
	resultRedactionTruncate = "truncate"
	resultRedactionHash     = "hash"

	sessionGrantTTL = 4 * time.Hour
)

type CommandProfile struct {
	Risk                 RiskLevel `json:"risk"`
	Interactive          bool      `json:"interactive"`
	Mutating             bool      `json:"mutating"`
	SessionGrantEligible bool      `json:"session_grant_eligible"`
	RequireTargetBinding bool      `json:"require_target_binding"`
	ExecSanitizer        string    `json:"exec_sanitizer,omitempty"`
	ResultRedaction      string    `json:"result_redaction,omitempty"`
	DefaultTimeoutSec    int       `json:"default_timeout_sec"`
}

var defaultCommandProfiles = map[string]CommandProfile{
	"screen.snapshot": {
		Risk:              RiskLow,
		DefaultTimeoutSec: 30,
	},
	"window.list": {
		Risk:              RiskLow,
		DefaultTimeoutSec: 15,
	},
	"ui.inspect": {
		Risk:              RiskLow,
		DefaultTimeoutSec: 30,
	},
	"fs.read": {
		Risk:              RiskLow,
		DefaultTimeoutSec: 20,
	},
	"fs.list": {
		Risk:              RiskLow,
		DefaultTimeoutSec: 20,
	},
	"wsl.fs.read": {
		Risk:              RiskLow,
		DefaultTimeoutSec: 20,
	},
	"wsl.fs.list": {
		Risk:              RiskLow,
		DefaultTimeoutSec: 20,
	},
	"window.focus": {
		Risk:                 RiskMedium,
		Interactive:          true,
		Mutating:             true,
		SessionGrantEligible: true,
		RequireTargetBinding: true,
		DefaultTimeoutSec:    20,
	},
	"ui.focus": {
		Risk:                 RiskMedium,
		Interactive:          true,
		Mutating:             true,
		SessionGrantEligible: true,
		RequireTargetBinding: true,
		DefaultTimeoutSec:    20,
	},
	"input.mouse.click": {
		Risk:                 RiskHigh,
		Interactive:          true,
		Mutating:             true,
		SessionGrantEligible: true,
		RequireTargetBinding: true,
		DefaultTimeoutSec:    20,
	},
	"input.mouse.double_click": {
		Risk:                 RiskHigh,
		Interactive:          true,
		Mutating:             true,
		SessionGrantEligible: true,
		RequireTargetBinding: true,
		DefaultTimeoutSec:    20,
	},
	"input.mouse.right_click": {
		Risk:                 RiskHigh,
		Interactive:          true,
		Mutating:             true,
		SessionGrantEligible: true,
		RequireTargetBinding: true,
		DefaultTimeoutSec:    20,
	},
	"input.keyboard.type": {
		Risk:                 RiskHigh,
		Interactive:          true,
		Mutating:             true,
		SessionGrantEligible: true,
		RequireTargetBinding: true,
		ResultRedaction:      resultRedactionHash,
		DefaultTimeoutSec:    20,
	},
	"input.keyboard.key": {
		Risk:                 RiskHigh,
		Interactive:          true,
		Mutating:             true,
		SessionGrantEligible: true,
		RequireTargetBinding: true,
		ResultRedaction:      resultRedactionTruncate,
		DefaultTimeoutSec:    20,
	},
	"input.keyboard.hotkey": {
		Risk:                 RiskHigh,
		Interactive:          true,
		Mutating:             true,
		SessionGrantEligible: true,
		RequireTargetBinding: true,
		ResultRedaction:      resultRedactionTruncate,
		DefaultTimeoutSec:    20,
	},
	"input.mouse.drag": {
		Risk:              RiskHigh,
		Interactive:       true,
		Mutating:          true,
		DefaultTimeoutSec: 20,
	},
	"system.exec": {
		Risk:                 RiskHigh,
		Mutating:             true,
		SessionGrantEligible: true,
		ExecSanitizer:        execSanitizerStructured,
		ResultRedaction:      resultRedactionHash,
		DefaultTimeoutSec:    30,
	},
	"fs.write": {
		Risk:                 RiskHigh,
		Mutating:             true,
		SessionGrantEligible: true,
		ResultRedaction:      resultRedactionHash,
		DefaultTimeoutSec:    20,
	},
	"wsl.exec": {
		Risk:                 RiskHigh,
		Mutating:             true,
		SessionGrantEligible: true,
		ExecSanitizer:        execSanitizerStructured,
		ResultRedaction:      resultRedactionHash,
		DefaultTimeoutSec:    30,
	},
	"wsl.fs.write": {
		Risk:                 RiskHigh,
		Mutating:             true,
		SessionGrantEligible: true,
		ResultRedaction:      resultRedactionHash,
		DefaultTimeoutSec:    20,
	},
}

var directAllowCapabilities = map[string]bool{
	"screen.snapshot": true,
	"window.list":     true,
	"ui.inspect":      true,
	"fs.read":         true,
	"fs.list":         true,
	"wsl.fs.read":     true,
	"wsl.fs.list":     true,
}

var blockedExecEnvKeys = map[string]bool{
	"PATH":         true,
	"Path":         true,
	"ComSpec":      true,
	"PSModulePath": true,
	"PATHEXT":      true,
}

var blockedExecArgPairs = map[string]map[string]bool{
	"cmd": {
		"/c": true,
		"/k": true,
	},
	"cmd.exe": {
		"/c": true,
		"/k": true,
	},
	"powershell": {
		"-command":        true,
		"-encodedcommand": true,
		"-c":              true,
	},
	"pwsh": {
		"-command":        true,
		"-encodedcommand": true,
		"-c":              true,
	},
	"python": {
		"-c": true,
	},
	"python3": {
		"-c": true,
	},
	"bash": {
		"-c": true,
	},
	"sh": {
		"-c": true,
	},
	"zsh": {
		"-c": true,
	},
	"fish": {
		"-c": true,
	},
}

func commandProfile(capability string) CommandProfile {
	if profile, ok := defaultCommandProfiles[capability]; ok {
		return profile
	}
	return CommandProfile{
		Risk:              RiskHigh,
		Mutating:          true,
		DefaultTimeoutSec: 20,
	}
}

func approvalMode(decision ApprovalDecision) ApprovalMode {
	if decision.Mode != "" {
		switch decision.Mode {
		case ApprovalModeAllowOnce, ApprovalModeAllowSession, ApprovalModeReject:
			return decision.Mode
		}
	}
	if decision.Approved {
		return ApprovalModeAllowOnce
	}
	return ApprovalModeReject
}

func normalizeNodeRequest(desc NodeDescriptor, req NodeCommandRequest, now time.Time) (NodeCommandRequest, CommandProfile, ApprovalBinding, error) {
	profile := commandProfile(req.Capability)
	req.Arguments = cloneArguments(req.Arguments)
	if req.TimeoutSec <= 0 && profile.DefaultTimeoutSec > 0 {
		req.TimeoutSec = profile.DefaultTimeoutSec
	}
	if req.ActionID == "" {
		req.ActionID = req.ID
	}

	binding := ApprovalBinding{
		SessionID:  req.SessionID,
		UserID:     req.UserID,
		NodeID:     req.NodeID,
		Capability: req.Capability,
		CreatedAt:  now.Unix(),
		ExpiresAt:  now.Add(sessionGrantTTL).Unix(),
	}

	switch req.Capability {
	case "system.exec":
		normalized, err := normalizeStructuredExecArgs(req.Arguments, "dir")
		if err != nil {
			return req, profile, binding, err
		}
		req.Arguments = normalized
		binding.ArgsDigest = digestJSON(map[string]interface{}{
			"command": normalized["command"],
			"args":    normalized["args"],
			"dir":     normalized["dir"],
			"env":     normalized["env"],
		})
	case "wsl.exec":
		normalized, err := normalizeStructuredExecArgs(req.Arguments, "cwd")
		if err != nil {
			return req, profile, binding, err
		}
		req.Arguments = normalized
		binding.ArgsDigest = digestJSON(map[string]interface{}{
			"command": normalized["command"],
			"args":    normalized["args"],
			"cwd":     normalized["cwd"],
			"env":     normalized["env"],
		})
	case "fs.write":
		target := strings.TrimSpace(stringArg(req.Arguments, "path"))
		if target == "" {
			return req, profile, binding, errors.New("fs.write requires path")
		}
		req.Arguments["path"] = filepath.Clean(target)
		binding.ArgsDigest = digestJSON(map[string]interface{}{
			"path":         req.Arguments["path"],
			"append":       boolArg(req.Arguments, "append"),
			"encoding":     strings.TrimSpace(stringArg(req.Arguments, "encoding")),
			"content_hash": shortHashString(stringArg(req.Arguments, "content")),
		})
	case "wsl.fs.write":
		target := normalizeApprovalPath(stringArg(req.Arguments, "path"))
		if target == "" {
			return req, profile, binding, errors.New("wsl.fs.write requires path")
		}
		req.Arguments["path"] = target
		binding.ArgsDigest = digestJSON(map[string]interface{}{
			"path":         req.Arguments["path"],
			"append":       boolArg(req.Arguments, "append"),
			"encoding":     strings.TrimSpace(stringArg(req.Arguments, "encoding")),
			"content_hash": shortHashString(stringArg(req.Arguments, "content")),
		})
	default:
		windowBinding := normalizeWindowBinding(req.Arguments)
		elementBinding := normalizeElementBinding(req.Arguments)
		binding.WindowBinding = windowBinding
		binding.ElementBinding = elementBinding
		binding.ArgsDigest = digestGUIArgs(req.Capability, req.Arguments)
	}

	if binding.WindowBinding == "" {
		binding.WindowBinding = normalizeWindowBinding(req.Arguments)
	}
	if binding.ElementBinding == "" {
		binding.ElementBinding = normalizeElementBinding(req.Arguments)
	}
	req.BindingHint = map[string]interface{}{
		"args_digest":     binding.ArgsDigest,
		"window_binding":  binding.WindowBinding,
		"element_binding": binding.ElementBinding,
	}
	if profile.RequireTargetBinding && !hasRequiredBinding(profile, binding) {
		profile.SessionGrantEligible = false
	}

	if approvalBypassedByMetadata(desc, req) {
		req.RequireApproval = false
	}
	return req, profile, binding, nil
}

func normalizeStructuredExecArgs(arguments map[string]interface{}, cwdKey string) (map[string]interface{}, error) {
	normalized := cloneArguments(arguments)
	command := strings.TrimSpace(stringArg(arguments, "command"))
	if command == "" {
		return nil, fmt.Errorf("%s requires command", cwdKeyToCapability(cwdKey))
	}
	if strings.Contains(command, " ") {
		return nil, errors.New("command must be a single executable; pass argv items via args")
	}
	args := normalizeStringSlice(stringSliceArg(arguments, "args"))
	if err := validateStructuredExec(command, args, mapArg(arguments, "env")); err != nil {
		return nil, err
	}
	normalized["command"] = command
	if len(args) > 0 {
		normalized["args"] = args
	} else {
		delete(normalized, "args")
	}
	cwd := strings.TrimSpace(stringArg(arguments, cwdKey))
	if cwd != "" {
		normalized[cwdKey] = cwd
	} else {
		delete(normalized, cwdKey)
	}
	if env := mapArg(arguments, "env"); len(env) > 0 {
		normalizedEnv := make(map[string]interface{}, len(env))
		keys := make([]string, 0, len(env))
		for key := range env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			normalizedEnv[key] = env[key]
		}
		normalized["env"] = normalizedEnv
	} else {
		delete(normalized, "env")
	}
	return normalized, nil
}

func cwdKeyToCapability(cwdKey string) string {
	switch cwdKey {
	case "cwd":
		return "wsl.exec"
	default:
		return "system.exec"
	}
}

func validateStructuredExec(command string, args []string, env map[string]string) error {
	base := strings.ToLower(filepath.Base(command))
	if blockedArgs, ok := blockedExecArgPairs[base]; ok && len(args) > 0 {
		first := strings.ToLower(strings.TrimSpace(args[0]))
		if blockedArgs[first] {
			return fmt.Errorf("%s with %s is blocked; pass an explicit executable argv instead", command, args[0])
		}
	}
	for key := range env {
		if blockedExecEnvKeys[key] {
			return fmt.Errorf("environment override %s is not allowed", key)
		}
	}
	return nil
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func approvalBypassedByMetadata(desc NodeDescriptor, req NodeCommandRequest) bool {
	return !shouldRequireNodeApproval(desc, req)
}

func requiresApprovalForRequest(req NodeCommandRequest, profile CommandProfile) bool {
	if !req.RequireApproval {
		return false
	}
	return !directAllowCapabilities[req.Capability]
}

func allowedApprovalModes(req NodeCommandRequest, profile CommandProfile, binding ApprovalBinding) []ApprovalMode {
	if !requiresApprovalForRequest(req, profile) {
		return nil
	}
	modes := []ApprovalMode{ApprovalModeAllowOnce}
	if profile.SessionGrantEligible && bindingEligibleForSession(req.Capability, binding) {
		modes = append(modes, ApprovalModeAllowSession)
	}
	return modes
}

func hasRequiredBinding(profile CommandProfile, binding ApprovalBinding) bool {
	if !profile.RequireTargetBinding {
		return true
	}
	return binding.WindowBinding != "" || binding.ElementBinding != ""
}

func bindingEligibleForSession(capability string, binding ApprovalBinding) bool {
	switch capability {
	case "window.focus":
		return hasStableWindowBinding(binding.WindowBinding)
	case "ui.focus", "input.mouse.click", "input.mouse.double_click", "input.mouse.right_click",
		"input.keyboard.type", "input.keyboard.key", "input.keyboard.hotkey":
		return hasStableWindowBinding(binding.WindowBinding) && hasStableElementBinding(binding.ElementBinding)
	case "system.exec", "wsl.exec", "fs.write", "wsl.fs.write":
		return binding.ArgsDigest != ""
	default:
		return false
	}
}

func hasStableWindowBinding(binding string) bool {
	if binding == "" {
		return false
	}
	return strings.Contains(binding, "handle=") || (strings.Contains(binding, "title=") && strings.Contains(binding, "process="))
}

func hasStableElementBinding(binding string) bool {
	if binding == "" {
		return false
	}
	return strings.Contains(binding, "path=") || strings.Contains(binding, "automation_id=")
}

func normalizeWindowBinding(args map[string]interface{}) string {
	window := map[string]string{
		"handle":  strings.TrimSpace(firstNonEmptyString(args, "window_handle", "handle")),
		"title":   strings.TrimSpace(firstNonEmptyString(args, "window_title", "title")),
		"process": strings.TrimSpace(firstNonEmptyString(args, "process_name")),
	}
	if element := elementArg(args); len(element) > 0 {
		if window["handle"] == "" {
			window["handle"] = strings.TrimSpace(firstNonEmptyString(element, "window_handle", "handle"))
		}
		if window["title"] == "" {
			window["title"] = strings.TrimSpace(firstNonEmptyString(element, "window_title", "title"))
		}
		if window["process"] == "" {
			window["process"] = strings.TrimSpace(firstNonEmptyString(element, "process_name"))
		}
	}
	parts := make([]string, 0, 3)
	if window["handle"] != "" {
		parts = append(parts, "handle="+window["handle"])
	}
	if window["title"] != "" {
		parts = append(parts, "title="+window["title"])
	}
	if window["process"] != "" {
		parts = append(parts, "process="+window["process"])
	}
	return strings.Join(parts, "|")
}

func normalizeElementBinding(args map[string]interface{}) string {
	element := elementArg(args)
	if len(element) == 0 {
		return ""
	}
	parts := make([]string, 0, 5)
	if value := strings.TrimSpace(firstNonEmptyString(element, "path")); value != "" {
		parts = append(parts, "path="+value)
	}
	if value := strings.TrimSpace(firstNonEmptyString(element, "automation_id")); value != "" {
		parts = append(parts, "automation_id="+value)
	}
	if value := strings.TrimSpace(firstNonEmptyString(element, "name")); value != "" {
		parts = append(parts, "name="+value)
	}
	if value := strings.TrimSpace(firstNonEmptyString(element, "role", "control_type")); value != "" {
		parts = append(parts, "role="+value)
	}
	if value := strings.TrimSpace(firstNonEmptyString(element, "class_name")); value != "" {
		parts = append(parts, "class="+value)
	}
	return strings.Join(parts, "|")
}

func digestGUIArgs(capability string, args map[string]interface{}) string {
	payload := map[string]interface{}{
		"capability": capability,
	}
	switch capability {
	case "input.keyboard.type":
		payload["text_hash"] = shortHashString(stringArg(args, "text"))
	case "input.keyboard.key":
		payload["key"] = strings.TrimSpace(firstNonEmptyString(args, "key"))
	case "input.keyboard.hotkey":
		payload["keys"] = normalizeStringSlice(stringSliceArg(args, "keys"))
	case "input.mouse.click", "input.mouse.double_click", "input.mouse.right_click":
		payload["button"] = strings.TrimSpace(firstNonEmptyString(args, "button"))
		payload["clicks"] = intArg(args, "clicks", 0)
		payload["x"] = stringArg(args, "x")
		payload["y"] = stringArg(args, "y")
	case "input.mouse.drag":
		payload["from_x"] = stringArg(args, "from_x")
		payload["from_y"] = stringArg(args, "from_y")
		payload["x"] = stringArg(args, "x")
		payload["y"] = stringArg(args, "y")
	case "window.focus":
		payload["window"] = normalizeWindowBinding(args)
	case "ui.focus":
		payload["element"] = normalizeElementBinding(args)
	case "screen.snapshot":
		payload["scope"] = strings.TrimSpace(stringArg(args, "scope"))
	}
	if element := normalizeElementBinding(args); element != "" {
		payload["element_binding"] = element
	}
	if window := normalizeWindowBinding(args); window != "" {
		payload["window_binding"] = window
	}
	if len(payload) <= 1 {
		return ""
	}
	return digestJSON(payload)
}

func digestJSON(value interface{}) string {
	if value == nil {
		return ""
	}
	content, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha1.Sum(content)
	return hex.EncodeToString(sum[:])
}

func shortHashString(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	sum := sha1.Sum([]byte(trimmed))
	return hex.EncodeToString(sum[:8])
}
