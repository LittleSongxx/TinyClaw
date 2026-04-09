package tooling

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/node"
)

const (
	toolNodeListDevices    = "node_list_devices"
	toolNodeSystemExec     = "node_system_exec"
	toolNodeFSList         = "node_fs_list"
	toolNodeFSRead         = "node_fs_read"
	toolNodeFSWrite        = "node_fs_write"
	toolNodeScreenShot     = "node_screen_snapshot"
	toolNodeBrowserOpen    = "node_browser_open"
	toolNodeAppLaunch      = "node_app_launch"
	toolNodeKeyboardType   = "node_keyboard_type"
	toolNodeKeyboardKey    = "node_keyboard_key"
	toolNodeKeyboardHotkey = "node_keyboard_hotkey"
	toolNodeMouseMove      = "node_mouse_move"
	toolNodeMouseClick     = "node_mouse_click"
	toolNodeMouseDrag      = "node_mouse_drag"
	toolNodeWindowList     = "node_window_list"
	toolNodeWindowFocus    = "node_window_focus"
	toolNodeUIInspect      = "node_ui_inspect"
	toolNodeUIFind         = "node_ui_find"
	toolNodeUIFocus        = "node_ui_focus"
	toolNodeWSLExec        = "node_wsl_exec"
	toolNodeWSLFSList      = "node_wsl_fs_list"
	toolNodeWSLFSRead      = "node_wsl_fs_read"
	toolNodeWSLFSWrite     = "node_wsl_fs_write"
	argNodeID              = "node_id"
	argTimeoutSec          = "timeout_sec"
)

var capabilityToolMap = map[string]string{
	"system.exec":              toolNodeSystemExec,
	"fs.list":                  toolNodeFSList,
	"fs.read":                  toolNodeFSRead,
	"fs.write":                 toolNodeFSWrite,
	"screen.snapshot":          toolNodeScreenShot,
	"browser.open":             toolNodeBrowserOpen,
	"app.launch":               toolNodeAppLaunch,
	"input.keyboard.type":      toolNodeKeyboardType,
	"input.keyboard.key":       toolNodeKeyboardKey,
	"input.keyboard.hotkey":    toolNodeKeyboardHotkey,
	"input.mouse.move":         toolNodeMouseMove,
	"input.mouse.click":        toolNodeMouseClick,
	"input.mouse.double_click": toolNodeMouseClick,
	"input.mouse.right_click":  toolNodeMouseClick,
	"input.mouse.drag":         toolNodeMouseDrag,
	"window.list":              toolNodeWindowList,
	"window.focus":             toolNodeWindowFocus,
	"ui.inspect":               toolNodeUIInspect,
	"ui.find":                  toolNodeUIFind,
	"ui.focus":                 toolNodeUIFocus,
	"wsl.exec":                 toolNodeWSLExec,
	"wsl.fs.list":              toolNodeWSLFSList,
	"wsl.fs.read":              toolNodeWSLFSRead,
	"wsl.fs.write":             toolNodeWSLFSWrite,
}

var toolCapabilityMap = map[string]string{
	toolNodeSystemExec:     "system.exec",
	toolNodeFSList:         "fs.list",
	toolNodeFSRead:         "fs.read",
	toolNodeFSWrite:        "fs.write",
	toolNodeScreenShot:     "screen.snapshot",
	toolNodeBrowserOpen:    "browser.open",
	toolNodeAppLaunch:      "app.launch",
	toolNodeKeyboardType:   "input.keyboard.type",
	toolNodeKeyboardKey:    "input.keyboard.key",
	toolNodeKeyboardHotkey: "input.keyboard.hotkey",
	toolNodeMouseMove:      "input.mouse.move",
	toolNodeMouseClick:     "input.mouse.click",
	toolNodeMouseDrag:      "input.mouse.drag",
	toolNodeWindowList:     "window.list",
	toolNodeWindowFocus:    "window.focus",
	toolNodeUIInspect:      "ui.inspect",
	toolNodeUIFind:         "ui.find",
	toolNodeUIFocus:        "ui.focus",
	toolNodeWSLExec:        "wsl.exec",
	toolNodeWSLFSList:      "wsl.fs.list",
	toolNodeWSLFSRead:      "wsl.fs.read",
	toolNodeWSLFSWrite:     "wsl.fs.write",
}

type NodeProvider struct {
	nodes node.Broker
}

func NewNodeProvider(nodes node.Broker) *NodeProvider {
	return &NodeProvider{nodes: nodes}
}

func (p *NodeProvider) Name() string {
	return "pc-node-tools"
}

func (p *NodeProvider) ChatVisible() bool {
	return true
}

func (p *NodeProvider) ListTools(ctx context.Context) ([]ToolSpec, error) {
	if p == nil {
		return nil, nil
	}

	items := []ToolSpec{nodeListToolSpec()}
	if p.nodes == nil {
		return items, nil
	}

	seen := make(map[string]bool)
	for _, current := range p.nodes.ListNodes(ctx) {
		for _, capability := range current.Capabilities {
			toolName, ok := capabilityToolMap[capability.Name]
			if !ok || seen[toolName] {
				continue
			}
			spec, ok := buildNodeToolSpec(capability.Name)
			if !ok {
				continue
			}
			items = append(items, spec)
			seen[toolName] = true
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (p *NodeProvider) Supports(name string) bool {
	if name == toolNodeListDevices {
		return true
	}
	_, ok := toolCapabilityMap[name]
	return ok
}

func (p *NodeProvider) ExecuteTool(ctx context.Context, call ToolInvocation) (*ToolResult, error) {
	if call.Name == toolNodeListDevices {
		return p.listDevices(ctx), nil
	}
	if p == nil || p.nodes == nil {
		return &ToolResult{
			Name:        call.Name,
			Output:      `{"success":false,"error":"node broker is not initialized"}`,
			StartedAt:   time.Now().Unix(),
			CompletedAt: time.Now().Unix(),
		}, nil
	}

	capability, ok := toolCapabilityMap[call.Name]
	if !ok {
		return nil, ErrToolProviderNotFound
	}

	arguments := cloneArguments(call.Arguments)
	req := node.NodeCommandRequest{
		NodeID:          stringArgument(arguments, argNodeID, call.NodeID),
		SessionID:       call.SessionID,
		UserID:          call.UserID,
		Capability:      capability,
		Arguments:       arguments,
		TimeoutSec:      intArgument(arguments, argTimeoutSec),
		RequireApproval: !conf.IsPrivilegedUser(call.UserID),
	}
	delete(req.Arguments, argNodeID)
	delete(req.Arguments, argTimeoutSec)

	startedAt := time.Now().Unix()
	result, err := p.nodes.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Name:        call.Name,
		Output:      formatNodeResult(result),
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
	}, nil
}

func (p *NodeProvider) listDevices(ctx context.Context) *ToolResult {
	items := make([]map[string]interface{}, 0)
	if p != nil && p.nodes != nil {
		for _, current := range p.nodes.ListNodes(ctx) {
			capabilities := make([]string, 0, len(current.Capabilities))
			for _, capability := range current.Capabilities {
				capabilities = append(capabilities, capability.Name)
			}
			sort.Strings(capabilities)
			items = append(items, map[string]interface{}{
				"id":           current.ID,
				"name":         current.Name,
				"platform":     current.Platform,
				"hostname":     current.Hostname,
				"version":      current.Version,
				"metadata":     current.Metadata,
				"last_seen_at": current.LastSeenAt,
				"capabilities": capabilities,
			})
		}
	}

	output, _ := json.Marshal(map[string]interface{}{
		"nodes": items,
		"count": len(items),
	})
	now := time.Now().Unix()
	return &ToolResult{
		Name:        toolNodeListDevices,
		Output:      string(output),
		StartedAt:   now,
		CompletedAt: now,
	}
}

func buildNodeToolSpec(capability string) (ToolSpec, bool) {
	switch capability {
	case "system.exec":
		return ToolSpec{
			Name:        toolNodeSystemExec,
			Category:    CategoryNode,
			Description: "Execute a real shell command on a paired PC node. Use this for Windows PowerShell or command prompt tasks on the user's computer.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID:     nodeIDProperty(),
					"command":     stringProperty("The executable to run on the paired PC, such as powershell, cmd, python or notepad.exe."),
					"args":        arrayProperty("Command arguments passed to the executable in order."),
					"dir":         stringProperty("Optional working directory on the paired PC."),
					"env":         mapProperty("Optional environment variables to set for the command."),
					argTimeoutSec: intProperty("Optional command timeout in seconds."),
				},
				"command",
			),
		}, true
	case "fs.list":
		return ToolSpec{
			Name:        toolNodeFSList,
			Category:    CategoryNode,
			Description: "List files or folders on a paired PC node. Use this before reading or writing when you need to inspect a directory.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID: nodeIDProperty(),
					"path":    stringProperty("Directory path on the paired PC. Defaults to the current directory when omitted."),
				},
			),
		}, true
	case "fs.read":
		return ToolSpec{
			Name:        toolNodeFSRead,
			Category:    CategoryNode,
			Description: "Read a file from a paired PC node.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID: nodeIDProperty(),
					"path":    stringProperty("Absolute or relative file path on the paired PC."),
				},
				"path",
			),
		}, true
	case "fs.write":
		return ToolSpec{
			Name:        toolNodeFSWrite,
			Category:    CategoryNode,
			Description: "Write a file on a paired PC node. Use this only when the user clearly wants a file created or changed.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID:  nodeIDProperty(),
					"path":     stringProperty("Absolute or relative file path on the paired PC."),
					"content":  stringProperty("Text content to write. Use base64 only when encoding is explicitly set to base64."),
					"append":   boolProperty("Append instead of replacing the file."),
					"encoding": stringProperty("Optional content encoding, such as base64."),
				},
				"path",
				"content",
			),
		}, true
	case "screen.snapshot":
		return ToolSpec{
			Name:        toolNodeScreenShot,
			Category:    CategoryNode,
			Description: "Capture a screenshot from a real paired PC node. Use scope active_window when the user asks about the current window, this app, or a named desktop application.",
			InputSchema: objectSchema(
				mergeProperties(
					map[string]interface{}{
						argNodeID: nodeIDProperty(),
						"scope":   stringProperty("Optional screenshot scope. Use virtual_desktop by default. Supported values include virtual_desktop, primary, and active_window."),
					},
					windowSelectorProperties(),
				),
			),
		}, true
	case "browser.open":
		return ToolSpec{
			Name:        toolNodeBrowserOpen,
			Category:    CategoryNode,
			Description: "Open a URL in the default browser on a paired PC node.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID: nodeIDProperty(),
					"url":     stringProperty("HTTP or HTTPS URL to open in the user's browser."),
				},
				"url",
			),
		}, true
	case "app.launch":
		return ToolSpec{
			Name:        toolNodeAppLaunch,
			Category:    CategoryNode,
			Description: "Launch a desktop application on a paired PC node, such as notepad.exe on Windows.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID: nodeIDProperty(),
					"command": stringProperty("Application executable or command to launch, such as notepad.exe."),
					"path":    stringProperty("Alternative application path when command is not provided."),
					"args":    arrayProperty("Optional application arguments."),
					"dir":     stringProperty("Optional working directory."),
				},
			),
		}, true
	case "input.keyboard.type":
		return ToolSpec{
			Name:        toolNodeKeyboardType,
			Category:    CategoryNode,
			Description: "Type text into the active desktop window on the paired PC node. When possible, pass an element locator so the node can target a specific input control. This requires user confirmation before execution.",
			InputSchema: objectSchema(
				mergeProperties(
					map[string]interface{}{
						argNodeID: nodeIDProperty(),
						"text":    stringProperty("Text to type into the currently focused desktop window or a located UI element."),
					},
					elementLocatorProperties(),
				),
				"text",
			),
		}, true
	case "input.keyboard.key":
		return ToolSpec{
			Name:        toolNodeKeyboardKey,
			Category:    CategoryNode,
			Description: "Press a key such as Enter, Tab or Escape on the paired PC node. This requires user confirmation before execution.",
			InputSchema: objectSchema(
				mergeProperties(
					map[string]interface{}{
						argNodeID: nodeIDProperty(),
						"key":     stringProperty("Key to press, such as ENTER, TAB, ESC, LEFT or F5."),
						"repeat":  intProperty("Optional repeat count."),
					},
					elementLocatorProperties(),
				),
				"key",
			),
		}, true
	case "input.keyboard.hotkey":
		return ToolSpec{
			Name:        toolNodeKeyboardHotkey,
			Category:    CategoryNode,
			Description: "Trigger a hotkey like Ctrl+S or Alt+Tab on the paired PC node. This requires user confirmation before execution.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID: nodeIDProperty(),
					"keys":    arrayProperty("Ordered keys for the hotkey, such as [\"CTRL\", \"S\"] or [\"ALT\", \"TAB\"]."),
				},
				"keys",
			),
		}, true
	case "input.mouse.move":
		return ToolSpec{
			Name:        toolNodeMouseMove,
			Category:    CategoryNode,
			Description: "Move the mouse cursor on the paired PC node.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID: nodeIDProperty(),
					"x":       intProperty("Target X coordinate in screen pixels."),
					"y":       intProperty("Target Y coordinate in screen pixels."),
				},
				"x",
				"y",
			),
		}, true
	case "input.mouse.click", "input.mouse.double_click", "input.mouse.right_click":
		return ToolSpec{
			Name:        toolNodeMouseClick,
			Category:    CategoryNode,
			Description: "Click on the paired PC node. Prefer an element locator so the node can click the target control center, and fall back to x/y only when there is no stable element. This requires user confirmation before execution.",
			InputSchema: objectSchema(
				mergeProperties(
					map[string]interface{}{
						argNodeID: nodeIDProperty(),
						"x":       intProperty("Target X coordinate in screen pixels. Only use this when there is no stable UI element locator."),
						"y":       intProperty("Target Y coordinate in screen pixels. Only use this when there is no stable UI element locator."),
						"button":  stringProperty("Mouse button. Use left by default or right when needed."),
						"clicks":  intProperty("Optional click count. Use 2 for double click."),
					},
					elementLocatorProperties(),
				),
			),
		}, true
	case "input.mouse.drag":
		return ToolSpec{
			Name:        toolNodeMouseDrag,
			Category:    CategoryNode,
			Description: "Drag the mouse from one point to another on the paired PC node. This requires user confirmation before execution.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID:     nodeIDProperty(),
					"from_x":      intProperty("Drag starting X coordinate in screen pixels."),
					"from_y":      intProperty("Drag starting Y coordinate in screen pixels."),
					"x":           intProperty("Drag destination X coordinate in screen pixels."),
					"y":           intProperty("Drag destination Y coordinate in screen pixels."),
					"duration_ms": intProperty("Optional drag duration in milliseconds."),
					"steps":       intProperty("Optional number of interpolation steps during drag."),
				},
				"from_x",
				"from_y",
				"x",
				"y",
			),
		}, true
	case "window.list":
		return ToolSpec{
			Name:        toolNodeWindowList,
			Category:    CategoryNode,
			Description: "List top-level desktop windows on the paired PC node so the agent can decide what to focus next.",
			InputSchema: objectSchema(map[string]interface{}{
				argNodeID: nodeIDProperty(),
			}),
		}, true
	case "window.focus":
		return ToolSpec{
			Name:        toolNodeWindowFocus,
			Category:    CategoryNode,
			Description: "Focus a top-level desktop window on the paired PC node using its title, process name or handle.",
			InputSchema: objectSchema(
				mergeProperties(
					map[string]interface{}{
						argNodeID: nodeIDProperty(),
						"title":   stringProperty("Window title or title fragment to match."),
						"handle":  stringProperty("Optional native window handle."),
					},
					windowSelectorProperties(),
				),
			),
		}, true
	case "ui.inspect":
		return ToolSpec{
			Name:        toolNodeUIInspect,
			Category:    CategoryNode,
			Description: "Inspect desktop UI elements. Use mode window_tree to inspect the active window subtree, mode focused for the focused control, or mode point for a specific screen coordinate.",
			InputSchema: objectSchema(
				mergeProperties(
					map[string]interface{}{
						argNodeID: nodeIDProperty(),
						"mode":    stringProperty("Inspection mode: window_tree, focused, or point. Use window_tree by default."),
						"depth":   intProperty("Optional UI tree depth. Defaults to 4 for window_tree mode."),
						"x":       intProperty("Optional X coordinate in screen pixels for point mode."),
						"y":       intProperty("Optional Y coordinate in screen pixels for point mode."),
					},
					windowSelectorProperties(),
				),
			),
		}, true
	case "ui.find":
		return ToolSpec{
			Name:        toolNodeUIFind,
			Category:    CategoryNode,
			Description: "Find controls inside the current or specified desktop window. Prefer this before clicking or typing into buttons, text boxes, checkboxes, menus, or other UI elements.",
			InputSchema: objectSchema(
				mergeProperties(
					map[string]interface{}{
						argNodeID:       nodeIDProperty(),
						"automation_id": stringProperty("Optional automation id to match."),
						"name":          stringProperty("Optional element name or label to match."),
						"role":          stringProperty("Optional control role such as button, edit, checkbox, menuitem, or ControlType.Button."),
						"class_name":    stringProperty("Optional class name to match."),
						"path":          stringProperty("Optional stable UI tree path such as 0.2.1."),
						"index":         intProperty("Optional zero-based match index after filtering."),
						"exact":         boolProperty("When true, use exact string matching instead of fuzzy contains matching."),
						"max_results":   intProperty("Optional maximum number of matched elements to return."),
						"depth":         intProperty("Optional search depth. Defaults to 6."),
					},
					windowSelectorProperties(),
				),
			),
		}, true
	case "ui.focus":
		return ToolSpec{
			Name:        toolNodeUIFocus,
			Category:    CategoryNode,
			Description: "Focus a specific UI Automation element inside a desktop window before follow-up actions.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID: nodeIDProperty(),
					"element": objectProperty("Locator for the target desktop UI element.", elementLocatorProperties()),
				},
				"element",
			),
		}, true
	case "wsl.exec":
		return ToolSpec{
			Name:        toolNodeWSLExec,
			Category:    CategoryNode,
			Description: "Execute an explicit argv command inside a WSL virtual node. Put the executable in command and the remaining argv items in args. Do not use shell wrappers like bash -c.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID:     nodeIDProperty(),
					"command":     stringProperty("Executable to run inside the selected WSL distro, such as git, ls, npm, go, or python."),
					"args":        arrayProperty("Optional argv items passed to the executable in order. Use this instead of shell wrappers."),
					"cwd":         stringProperty("Optional Linux working directory. When omitted, the node uses the distro default directory or the distro home."),
					"env":         mapProperty("Optional environment variables to export before running the command."),
					argTimeoutSec: intProperty("Optional command timeout in seconds."),
				},
				"command",
			),
		}, true
	case "wsl.fs.list":
		return ToolSpec{
			Name:        toolNodeWSLFSList,
			Category:    CategoryNode,
			Description: "List files or folders inside the selected WSL virtual node.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID: nodeIDProperty(),
					"path":    stringProperty("Linux directory path inside the selected WSL distro. Defaults to . when omitted."),
				},
			),
		}, true
	case "wsl.fs.read":
		return ToolSpec{
			Name:        toolNodeWSLFSRead,
			Category:    CategoryNode,
			Description: "Read a file from the selected WSL virtual node.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID: nodeIDProperty(),
					"path":    stringProperty("Linux file path inside the selected WSL distro."),
				},
				"path",
			),
		}, true
	case "wsl.fs.write":
		return ToolSpec{
			Name:        toolNodeWSLFSWrite,
			Category:    CategoryNode,
			Description: "Write a file inside the selected WSL virtual node. Use this when the user wants a Linux-side file created or changed.",
			InputSchema: objectSchema(
				map[string]interface{}{
					argNodeID:  nodeIDProperty(),
					"path":     stringProperty("Linux file path inside the selected WSL distro."),
					"content":  stringProperty("Text content to write. Use base64 only when encoding is explicitly set to base64."),
					"append":   boolProperty("Append instead of replacing the file."),
					"encoding": stringProperty("Optional content encoding, such as base64."),
				},
				"path",
				"content",
			),
		}, true
	default:
		return ToolSpec{}, false
	}
}

func nodeListToolSpec() ToolSpec {
	return ToolSpec{
		Name:        toolNodeListDevices,
		Category:    CategoryNode,
		Description: "List the currently connected PC nodes and their capabilities before choosing which real device to control.",
		InputSchema: objectSchema(nil),
	}
}

func objectSchema(properties map[string]interface{}, required ...string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
	if properties != nil {
		schema["properties"] = properties
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func nodeIDProperty() map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": "Optional node id. Leave it empty unless the user specified a particular device, and the gateway will choose a compatible online PC node automatically.",
	}
}

func stringProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": description,
	}
}

func arrayProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": description,
		"items": map[string]interface{}{
			"type": "string",
		},
	}
}

func mapProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":                 "object",
		"description":          description,
		"additionalProperties": map[string]interface{}{"type": "string"},
	}
}

func boolProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": description,
	}
}

func intProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": description,
	}
}

func objectProperty(description string, properties map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"description": description,
		"properties":  properties,
	}
}

func cloneArguments(arguments map[string]interface{}) map[string]interface{} {
	if len(arguments) == 0 {
		return map[string]interface{}{}
	}
	cloned := make(map[string]interface{}, len(arguments))
	for key, value := range arguments {
		cloned[key] = value
	}
	return cloned
}

func stringArgument(arguments map[string]interface{}, key, fallback string) string {
	if arguments != nil {
		if raw, ok := arguments[key]; ok {
			if value, ok := raw.(string); ok {
				return strings.TrimSpace(value)
			}
		}
	}
	return fallback
}

func intArgument(arguments map[string]interface{}, key string) int {
	if arguments == nil {
		return 0
	}
	raw, ok := arguments[key]
	if !ok {
		return 0
	}
	switch value := raw.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func formatNodeResult(result *node.NodeCommandResult) string {
	if result == nil {
		return `{"success":false,"error":"empty node response"}`
	}

	if pending, ok := result.Data["pending_approval"].(bool); ok && pending {
		payload := map[string]interface{}{
			"pending_approval": true,
			"approval_id":      result.Data["approval_id"],
			"summary":          result.Data["summary"],
			"capability":       result.Capability,
			"arguments":        result.Data["arguments"],
			"approval_modes":   result.Data["approval_modes"],
			"session_id":       result.Data["session_id"],
			"node_id":          result.NodeID,
		}
		content, _ := json.Marshal(payload)
		return string(content)
	}

	if result.Capability == "screen.snapshot" {
		if base64Data, ok := result.Data["base64"].(string); ok && base64Data != "" {
			mimeType, _ := result.Data["mime_type"].(string)
			imagePayload, _ := json.Marshal(map[string]interface{}{
				"type":     "image",
				"data":     base64Data,
				"mimeType": mimeType,
				"meta": map[string]interface{}{
					"scope":         result.Data["scope"],
					"width":         result.Data["width"],
					"height":        result.Data["height"],
					"display_count": result.Data["display_count"],
					"window":        result.Data["window"],
				},
			})
			return string(imagePayload)
		}
	}

	payload := map[string]interface{}{
		"success":    result.Success,
		"node_id":    result.NodeID,
		"capability": result.Capability,
	}
	if result.Output != "" {
		payload["output"] = result.Output
	}
	if result.Error != "" {
		payload["error"] = result.Error
	}
	if len(result.Data) > 0 {
		payload["data"] = result.Data
	}

	content, err := json.Marshal(payload)
	if err != nil {
		if result.Output != "" {
			return result.Output
		}
		return `{"success":false,"error":"failed to serialize node response"}`
	}
	return string(content)
}

func mergeProperties(groups ...map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for _, group := range groups {
		for key, value := range group {
			merged[key] = value
		}
	}
	return merged
}

func windowSelectorProperties() map[string]interface{} {
	return map[string]interface{}{
		"window_handle": stringProperty("Optional native window handle."),
		"window_title":  stringProperty("Optional window title or title fragment to match."),
		"process_name":  stringProperty("Optional process name such as notepad or msedge."),
	}
}

func elementLocatorProperties() map[string]interface{} {
	return map[string]interface{}{
		"element": map[string]interface{}{
			"type":        "object",
			"description": "Optional UI element locator. Prefer passing this when you want to click or type into a specific control instead of using raw coordinates.",
			"properties": map[string]interface{}{
				"path":          stringProperty("Stable UI tree path such as 0.2.1."),
				"automation_id": stringProperty("Automation id of the target control."),
				"name":          stringProperty("Visible control name or label."),
				"role":          stringProperty("Control role such as button, edit, checkbox, menuitem, or ControlType.Button."),
				"class_name":    stringProperty("Native class name of the target control."),
				"index":         intProperty("Optional zero-based match index after filtering."),
				"exact":         boolProperty("Use exact matching for string fields."),
				"window_handle": stringProperty("Optional native window handle."),
				"window_title":  stringProperty("Optional window title or title fragment."),
				"process_name":  stringProperty("Optional process name of the target window."),
			},
		},
	}
}
