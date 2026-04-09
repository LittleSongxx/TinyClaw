package node

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type LocalDriver struct{}

func NewLocalDriver() *LocalDriver {
	return &LocalDriver{}
}

func (d *LocalDriver) Capabilities() []NodeCapability {
	return nodeCapabilitiesForRuntime()
}

func (d *LocalDriver) Execute(ctx context.Context, req NodeCommandRequest) (*NodeCommandResult, error) {
	startedAt := time.Now().Unix()
	result := &NodeCommandResult{
		ID:         req.ID,
		NodeID:     req.NodeID,
		Capability: req.Capability,
		StartedAt:  startedAt,
		Success:    false,
	}

	var err error
	switch req.Capability {
	case "system.exec":
		result, err = d.execCommand(ctx, req, startedAt)
	case "fs.list":
		result, err = d.listFiles(req, startedAt)
	case "fs.read":
		result, err = d.readFile(req, startedAt)
	case "fs.write":
		result, err = d.writeFile(req, startedAt)
	case "screen.snapshot":
		result, err = d.snapshot(ctx, req, startedAt)
	case "browser.open":
		result, err = d.openBrowser(ctx, req, startedAt)
	case "app.launch":
		result, err = d.launchApp(ctx, req, startedAt)
	case "input.keyboard.type":
		result, err = d.keyboardType(ctx, req, startedAt)
	case "input.keyboard.key":
		result, err = d.keyboardKey(ctx, req, startedAt)
	case "input.keyboard.hotkey":
		result, err = d.keyboardHotkey(ctx, req, startedAt)
	case "input.mouse.move":
		result, err = d.mouseMove(ctx, req, startedAt)
	case "input.mouse.click":
		result, err = d.mouseClick(ctx, req, startedAt, "click")
	case "input.mouse.double_click":
		result, err = d.mouseClick(ctx, req, startedAt, "double")
	case "input.mouse.right_click":
		result, err = d.mouseClick(ctx, req, startedAt, "right")
	case "input.mouse.drag":
		result, err = d.mouseDrag(ctx, req, startedAt)
	case "window.list":
		result, err = d.windowList(ctx, req, startedAt)
	case "window.focus":
		result, err = d.windowFocus(ctx, req, startedAt)
	case "ui.inspect":
		result, err = d.uiInspect(ctx, req, startedAt)
	case "ui.find":
		result, err = d.uiFind(ctx, req, startedAt)
	case "ui.focus":
		result, err = d.uiFocus(ctx, req, startedAt)
	default:
		err = errors.New("unsupported node capability")
	}

	if err != nil {
		if result == nil {
			result = &NodeCommandResult{
				ID:         req.ID,
				NodeID:     req.NodeID,
				Capability: req.Capability,
				StartedAt:  startedAt,
			}
		}
		result.Success = false
		result.Error = err.Error()
		result.CompletedAt = time.Now().Unix()
		return result, nil
	}

	result.Success = true
	if result.CompletedAt == 0 {
		result.CompletedAt = time.Now().Unix()
	}
	return result, nil
}

func (d *LocalDriver) execCommand(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	command := stringArg(req.Arguments, "command")
	if command == "" {
		return nil, errors.New("system.exec requires command")
	}

	timeout := req.TimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(runCtx, command, stringSliceArg(req.Arguments, "args")...)
	if dir := stringArg(req.Arguments, "dir"); dir != "" {
		cmd.Dir = dir
	}
	if envMap := mapArg(req.Arguments, "env"); len(envMap) > 0 {
		env := os.Environ()
		for key, val := range envMap {
			env = append(env, key+"="+val)
		}
		cmd.Env = env
	}

	output, err := cmd.CombinedOutput()
	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      string(output),
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"command": command,
			"args":    stringSliceArg(req.Arguments, "args"),
		},
	}, err
}

func (d *LocalDriver) listFiles(req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	target := stringArg(req.Arguments, "path")
	if target == "" {
		target = "."
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		return nil, err
	}

	items := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		items = append(items, map[string]interface{}{
			"name":    entry.Name(),
			"is_dir":  entry.IsDir(),
			"size":    info.Size(),
			"modtime": info.ModTime().Unix(),
		})
	}

	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"path":    target,
			"entries": items,
		},
	}, nil
}

func (d *LocalDriver) readFile(req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	target := stringArg(req.Arguments, "path")
	if target == "" {
		return nil, errors.New("fs.read requires path")
	}

	data, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}

	result := &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"path": target,
			"size": len(data),
		},
	}
	if utf8.Valid(data) {
		result.Output = string(data)
	} else {
		result.Data["base64"] = base64.StdEncoding.EncodeToString(data)
	}
	return result, nil
}

func (d *LocalDriver) writeFile(req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	target := stringArg(req.Arguments, "path")
	if target == "" {
		return nil, errors.New("fs.write requires path")
	}
	content := stringArg(req.Arguments, "content")
	appendMode := boolArg(req.Arguments, "append")
	encoding := stringArg(req.Arguments, "encoding")

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return nil, err
	}

	var data []byte
	if encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, err
		}
		data = decoded
	} else {
		data = []byte(content)
	}

	flag := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(target, flag, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return nil, err
	}

	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"path":   target,
			"append": appendMode,
			"size":   len(data),
		},
	}, nil
}

func (d *LocalDriver) snapshot(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	target, captureTarget, err := desktopTempFilePath("tinyclaw-screenshot-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".png")
	if err != nil {
		return nil, err
	}
	scope := stringArg(req.Arguments, "scope")
	if scope == "" {
		scope = "virtual_desktop"
	}
	meta, err := captureScreenshot(ctx, target, captureTarget, scope, req.Arguments)
	if err != nil {
		return nil, err
	}
	defer os.Remove(target)

	data, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}

	result := &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"mime_type":     "image/png",
			"base64":        base64.StdEncoding.EncodeToString(data),
			"path":          target,
			"scope":         meta.Scope,
			"width":         meta.Width,
			"height":        meta.Height,
			"display_count": meta.DisplayCount,
		},
	}
	if meta.Window != nil {
		result.Data["window"] = meta.Window
	}
	return result, nil
}

func (d *LocalDriver) openBrowser(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	url := stringArg(req.Arguments, "url")
	if url == "" {
		return nil, errors.New("browser.open requires url")
	}
	if err := openURL(ctx, url); err != nil {
		return nil, err
	}
	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "opened",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"url": url,
		},
	}, nil
}

func (d *LocalDriver) launchApp(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	command := stringArg(req.Arguments, "command")
	if command == "" {
		command = stringArg(req.Arguments, "path")
	}
	if command == "" {
		return nil, errors.New("app.launch requires command or path")
	}

	cmd := exec.CommandContext(ctx, command, stringSliceArg(req.Arguments, "args")...)
	if dir := stringArg(req.Arguments, "dir"); dir != "" {
		cmd.Dir = dir
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "launched",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"pid":     cmd.Process.Pid,
			"command": command,
		},
	}, nil
}

func (d *LocalDriver) keyboardType(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	text := stringArg(req.Arguments, "text")
	if text == "" {
		return nil, errors.New("input.keyboard.type requires text")
	}
	if !supportsWindowsDesktopAutomation() {
		return nil, errors.New("input.keyboard.type is only supported on windows for now")
	}
	if data, err := typeIntoWindowsElement(ctx, req.Arguments); err != nil {
		return nil, err
	} else if data != nil {
		return &NodeCommandResult{
			ID:          req.ID,
			NodeID:      req.NodeID,
			Capability:  req.Capability,
			Output:      "typed text",
			StartedAt:   startedAt,
			CompletedAt: time.Now().Unix(),
			Data:        data,
		}, nil
	}

	_, err := runPowerShell(ctx, `
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.SendKeys]::SendWait('`+escapePowerShellSingleQuoted(toSendKeysLiteral(text))+`')
`)
	if err != nil {
		return nil, err
	}

	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "typed text",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"text": text,
		},
	}, nil
}

func (d *LocalDriver) keyboardKey(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	key := stringArg(req.Arguments, "key")
	if key == "" {
		keys := stringSliceArg(req.Arguments, "keys")
		if len(keys) > 0 {
			key = keys[0]
		}
	}
	if key == "" {
		return nil, errors.New("input.keyboard.key requires key")
	}
	if !supportsWindowsDesktopAutomation() {
		return nil, errors.New("input.keyboard.key is only supported on windows for now")
	}
	var focusedElement map[string]interface{}
	if data, err := focusWindowsElement(ctx, req.Arguments); err != nil {
		return nil, err
	} else if data != nil {
		focusedElement = data
	}

	sendKey, err := toSendKeysToken(key)
	if err != nil {
		return nil, err
	}
	repeat := intArg(req.Arguments, "repeat", 1)
	if repeat < 1 {
		repeat = 1
	}
	_, err = runPowerShell(ctx, `
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms
for ($i = 0; $i -lt `+strconv.Itoa(repeat)+`; $i++) {
  [System.Windows.Forms.SendKeys]::SendWait('`+escapePowerShellSingleQuoted(sendKey)+`')
}
`)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"key":    key,
		"repeat": repeat,
	}
	if focusedElement != nil {
		data["element"] = focusedElement
	}
	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "pressed key",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data:        data,
	}, nil
}

func (d *LocalDriver) keyboardHotkey(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	keys := stringSliceArg(req.Arguments, "keys")
	if len(keys) == 0 {
		return nil, errors.New("input.keyboard.hotkey requires keys")
	}
	if !supportsWindowsDesktopAutomation() {
		return nil, errors.New("input.keyboard.hotkey is only supported on windows for now")
	}

	sendKeys, err := toHotkeySendKeys(keys)
	if err != nil {
		return nil, err
	}
	_, err = runPowerShell(ctx, `
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.SendKeys]::SendWait('`+escapePowerShellSingleQuoted(sendKeys)+`')
`)
	if err != nil {
		return nil, err
	}

	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "triggered hotkey",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"keys": keys,
		},
	}, nil
}

func (d *LocalDriver) mouseMove(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	if !supportsWindowsDesktopAutomation() {
		return nil, errors.New("input.mouse.move is only supported on windows for now")
	}
	x, okX := rawCoordinate(req.Arguments, "x")
	y, okY := rawCoordinate(req.Arguments, "y")
	if !okX || !okY {
		return nil, errors.New("input.mouse.move requires x and y")
	}
	_, err := runPowerShell(ctx, windowsUser32Preamble()+`
[TinyClawUser32]::SetCursorPos(`+strconv.Itoa(x)+`, `+strconv.Itoa(y)+`) | Out-Null
`)
	if err != nil {
		return nil, err
	}
	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "mouse moved",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"x": x,
			"y": y,
		},
	}, nil
}

func (d *LocalDriver) mouseClick(ctx context.Context, req NodeCommandRequest, startedAt int64, mode string) (*NodeCommandResult, error) {
	if !supportsWindowsDesktopAutomation() {
		return nil, errors.New("mouse click actions are only supported on windows for now")
	}
	button := strings.ToLower(stringArg(req.Arguments, "button"))
	if button == "" {
		if mode == "right" {
			button = "right"
		} else {
			button = "left"
		}
	}
	clickCount := 1
	if mode == "double" {
		clickCount = 2
	}
	if value := intArg(req.Arguments, "clicks", 0); value > 0 {
		clickCount = value
	}
	if data, err := clickWindowsElement(ctx, req.Arguments, button, clickCount); err != nil {
		return nil, err
	} else if data != nil {
		return &NodeCommandResult{
			ID:          req.ID,
			NodeID:      req.NodeID,
			Capability:  req.Capability,
			Output:      "mouse clicked",
			StartedAt:   startedAt,
			CompletedAt: time.Now().Unix(),
			Data:        data,
		}, nil
	}
	x, okX := rawCoordinate(req.Arguments, "x")
	y, okY := rawCoordinate(req.Arguments, "y")
	if !okX || !okY {
		return nil, errors.New("mouse click actions require x and y")
	}

	downFlag, upFlag, err := mouseButtonFlags(button)
	if err != nil {
		return nil, err
	}
	var script strings.Builder
	script.WriteString(windowsUser32Preamble())
	script.WriteString("\n[TinyClawUser32]::SetCursorPos(" + strconv.Itoa(x) + ", " + strconv.Itoa(y) + ") | Out-Null\n")
	for i := 0; i < clickCount; i++ {
		script.WriteString("[TinyClawUser32]::mouse_event(" + downFlag + ", 0, 0, 0, [UIntPtr]::Zero)\n")
		script.WriteString("[TinyClawUser32]::mouse_event(" + upFlag + ", 0, 0, 0, [UIntPtr]::Zero)\n")
	}
	_, err = runPowerShell(ctx, script.String())
	if err != nil {
		return nil, err
	}
	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "mouse clicked",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"x":      x,
			"y":      y,
			"button": button,
			"clicks": clickCount,
		},
	}, nil
}

func (d *LocalDriver) mouseDrag(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	if !supportsWindowsDesktopAutomation() {
		return nil, errors.New("input.mouse.drag is only supported on windows for now")
	}
	fromX, okFromX := rawCoordinate(req.Arguments, "from_x")
	fromY, okFromY := rawCoordinate(req.Arguments, "from_y")
	toX, okToX := rawCoordinate(req.Arguments, "x")
	toY, okToY := rawCoordinate(req.Arguments, "y")
	if !okFromX || !okFromY || !okToX || !okToY {
		return nil, errors.New("input.mouse.drag requires from_x, from_y, x and y")
	}
	duration := intArg(req.Arguments, "duration_ms", 300)
	if duration < 0 {
		duration = 0
	}
	steps := intArg(req.Arguments, "steps", 12)
	if steps < 1 {
		steps = 1
	}
	script := fmt.Sprintf(`%s
$fromX = %d
$fromY = %d
$toX = %d
$toY = %d
$steps = %d
$delay = %d
[TinyClawUser32]::SetCursorPos($fromX, $fromY) | Out-Null
[TinyClawUser32]::mouse_event([TinyClawUser32]::MOUSEEVENTF_LEFTDOWN, 0, 0, 0, [UIntPtr]::Zero)
for ($i = 1; $i -le $steps; $i++) {
  $nextX = [int]($fromX + (($toX - $fromX) * $i / $steps))
  $nextY = [int]($fromY + (($toY - $fromY) * $i / $steps))
  [TinyClawUser32]::SetCursorPos($nextX, $nextY) | Out-Null
  if ($delay -gt 0) { Start-Sleep -Milliseconds ([int]($delay / $steps)) }
}
[TinyClawUser32]::mouse_event([TinyClawUser32]::MOUSEEVENTF_LEFTUP, 0, 0, 0, [UIntPtr]::Zero)
`, windowsUser32Preamble(), fromX, fromY, toX, toY, steps, duration)
	_, err := runPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "mouse dragged",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"from_x":      fromX,
			"from_y":      fromY,
			"x":           toX,
			"y":           toY,
			"duration_ms": duration,
			"steps":       steps,
		},
	}, nil
}

func (d *LocalDriver) windowList(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	if !supportsWindowsDesktopAutomation() {
		return nil, errors.New("window.list is only supported on windows for now")
	}
	items, err := listWindowsDetailed(ctx)
	if err != nil {
		return nil, err
	}
	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "listed windows",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"windows": items,
			"count":   len(items),
		},
	}, nil
}

func (d *LocalDriver) windowFocus(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	if !supportsWindowsDesktopAutomation() {
		return nil, errors.New("window.focus is only supported on windows for now")
	}
	title := stringArg(req.Arguments, "title")
	processName := stringArg(req.Arguments, "process_name")
	handle := firstNonEmptyString(req.Arguments, "window_handle", "handle")
	if title == "" && processName == "" && handle == "" {
		return nil, errors.New("window.focus requires title, process_name or handle")
	}
	data, err := focusWindowDetailed(ctx, req.Arguments)
	if err != nil {
		return nil, err
	}
	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "focused window",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data:        data,
	}, nil
}

func (d *LocalDriver) uiInspect(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	if !supportsWindowsDesktopAutomation() {
		return nil, errors.New("ui.inspect is only supported on windows for now")
	}
	data, err := inspectWindowsUI(ctx, req.Arguments)
	if err != nil {
		return nil, err
	}
	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "inspected ui",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data:        data,
	}, nil
}

func (d *LocalDriver) uiFind(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	if !supportsWindowsDesktopAutomation() {
		return nil, errors.New("ui.find is only supported on windows for now")
	}
	data, err := findWindowsUI(ctx, req.Arguments)
	if err != nil {
		return nil, err
	}
	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "found ui elements",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data:        data,
	}, nil
}

func (d *LocalDriver) uiFocus(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	if !supportsWindowsDesktopAutomation() {
		return nil, errors.New("ui.focus is only supported on windows for now")
	}
	data, err := focusWindowsElement(ctx, req.Arguments)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, errors.New("ui.focus requires element locator")
	}
	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      "focused ui element",
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"element": data,
		},
	}, nil
}

type screenshotMeta struct {
	Scope        string      `json:"scope"`
	Width        int         `json:"width"`
	Height       int         `json:"height"`
	DisplayCount int         `json:"display_count"`
	Window       *windowMeta `json:"window,omitempty"`
}

func captureScreenshot(ctx context.Context, target, captureTarget, scope string, args map[string]interface{}) (*screenshotMeta, error) {
	if supportsWindowsDesktopAutomation() {
		return captureWindowsScreenshot(ctx, captureTarget, args)
	}
	switch runtime.GOOS {
	case "darwin":
		if err := exec.CommandContext(ctx, "screencapture", "-x", target).Run(); err != nil {
			return nil, err
		}
	default:
		commands := [][]string{
			{"gnome-screenshot", "-f", target},
			{"grim", target},
			{"scrot", target},
			{"import", "-window", "root", target},
		}
		for _, command := range commands {
			if _, err := exec.LookPath(command[0]); err == nil {
				if err := exec.CommandContext(ctx, command[0], command[1:]...).Run(); err != nil {
					return nil, err
				}
				return inferScreenshotMeta(target, scope)
			}
		}
		return nil, errors.New("no screenshot command found on this Linux host")
	}
	return inferScreenshotMeta(target, scope)
}

func openURL(ctx context.Context, url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "linux":
		if isWSLRuntime() {
			_, err := runPowerShell(ctx, `Start-Process '`+escapePowerShellSingleQuoted(url)+`'`)
			return err
		}
		return exec.CommandContext(ctx, "xdg-open", url).Start()
	case "darwin":
		return exec.CommandContext(ctx, "open", url).Start()
	default:
		return exec.CommandContext(ctx, "xdg-open", url).Start()
	}
}

func stringArg(args map[string]interface{}, key string) string {
	if len(args) == 0 {
		return ""
	}
	if raw, ok := args[key]; ok {
		switch val := raw.(type) {
		case string:
			return val
		case int:
			return strconv.Itoa(val)
		case int32:
			return strconv.FormatInt(int64(val), 10)
		case int64:
			return strconv.FormatInt(val, 10)
		case float64:
			return strconv.FormatInt(int64(val), 10)
		}
	}
	return ""
}

func stringSliceArg(args map[string]interface{}, key string) []string {
	if len(args) == 0 {
		return nil
	}
	raw, ok := args[key]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []string:
		return values
	case []interface{}:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if str, ok := value.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

func mapArg(args map[string]interface{}, key string) map[string]string {
	if len(args) == 0 {
		return nil
	}
	raw, ok := args[key]
	if !ok {
		return nil
	}
	switch val := raw.(type) {
	case map[string]string:
		return val
	case map[string]interface{}:
		out := make(map[string]string, len(val))
		for k, inner := range val {
			if str, ok := inner.(string); ok {
				out[k] = str
			}
		}
		return out
	default:
		return nil
	}
}

func boolArg(args map[string]interface{}, key string) bool {
	if len(args) == 0 {
		return false
	}
	raw, ok := args[key]
	if !ok {
		return false
	}
	val, ok := raw.(bool)
	return ok && val
}

func intArg(args map[string]interface{}, key string, fallback int) int {
	if len(args) == 0 {
		return fallback
	}
	raw, ok := args[key]
	if !ok {
		return fallback
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
		return fallback
	}
}

func rawCoordinate(args map[string]interface{}, key string) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	raw, ok := args[key]
	if !ok {
		return 0, false
	}
	switch value := raw.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}

func inferScreenshotMeta(target, scope string) (*screenshotMeta, error) {
	data, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if scope == "" {
		scope = "virtual_desktop"
	}
	return &screenshotMeta{
		Scope:        scope,
		Width:        cfg.Width,
		Height:       cfg.Height,
		DisplayCount: 1,
	}, nil
}

func runPowerShell(ctx context.Context, script string) ([]byte, error) {
	scriptPath, executablePath, err := desktopTempFilePath("tinyclaw-node-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".ps1")
	if err != nil {
		return nil, err
	}
	bootstrap := `[Console]::InputEncoding = [System.Text.UTF8Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::UTF8
$OutputEncoding = [System.Text.UTF8Encoding]::UTF8
` + "\n" + script
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte(bootstrap)...)
	if err := os.WriteFile(scriptPath, content, 0600); err != nil {
		return nil, err
	}
	defer os.Remove(scriptPath)

	executable, err := powerShellExecutable()
	if err != nil {
		return nil, err
	}
	output, err := exec.CommandContext(
		ctx,
		executable,
		"-NoProfile",
		"-STA",
		"-ExecutionPolicy",
		"Bypass",
		"-File",
		executablePath,
	).CombinedOutput()
	if err != nil {
		return nil, formatCommandError(err, output)
	}
	return bytes.TrimPrefix(output, []byte{0xEF, 0xBB, 0xBF}), nil
}

func decodePowerShellJSON(raw []byte, target interface{}) error {
	trimmed := bytes.TrimSpace(bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF}))
	if len(trimmed) == 0 {
		return errors.New("empty powershell json output")
	}
	return json.Unmarshal(trimmed, target)
}

func decodePowerShellJSONObject(raw []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := decodePowerShellJSON(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func decodePowerShellJSONArray(raw []byte) ([]map[string]interface{}, error) {
	trimmed := bytes.TrimSpace(bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF}))
	if len(trimmed) == 0 {
		return nil, nil
	}
	var list []map[string]interface{}
	if trimmed[0] == '{' {
		var single map[string]interface{}
		if err := json.Unmarshal(trimmed, &single); err != nil {
			return nil, err
		}
		return []map[string]interface{}{single}, nil
	}
	if err := json.Unmarshal(trimmed, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func escapePowerShellSingleQuoted(input string) string {
	return strings.ReplaceAll(input, "'", "''")
}

func windowsUser32Preamble() string {
	return `
$ErrorActionPreference = "Stop"
Add-Type @"
using System;
using System.Runtime.InteropServices;
public static class TinyClawUser32 {
  [DllImport("user32.dll")] public static extern bool SetCursorPos(int X, int Y);
  [DllImport("user32.dll")] public static extern void mouse_event(uint dwFlags, uint dx, uint dy, uint dwData, UIntPtr dwExtraInfo);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow);
  public const uint MOUSEEVENTF_LEFTDOWN = 0x0002;
  public const uint MOUSEEVENTF_LEFTUP = 0x0004;
  public const uint MOUSEEVENTF_RIGHTDOWN = 0x0008;
  public const uint MOUSEEVENTF_RIGHTUP = 0x0010;
}
"@
`
}

func mouseButtonFlags(button string) (string, string, error) {
	switch strings.ToLower(button) {
	case "", "left":
		return "[TinyClawUser32]::MOUSEEVENTF_LEFTDOWN", "[TinyClawUser32]::MOUSEEVENTF_LEFTUP", nil
	case "right":
		return "[TinyClawUser32]::MOUSEEVENTF_RIGHTDOWN", "[TinyClawUser32]::MOUSEEVENTF_RIGHTUP", nil
	default:
		return "", "", fmt.Errorf("unsupported mouse button: %s", button)
	}
}

func toSendKeysLiteral(input string) string {
	var builder strings.Builder
	for _, char := range input {
		switch char {
		case '+', '^', '%', '~', '(', ')', '[', ']', '{', '}':
			builder.WriteString("{")
			builder.WriteRune(char)
			builder.WriteString("}")
		case '\n':
			builder.WriteString("{ENTER}")
		case '\r':
		default:
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func toSendKeysToken(key string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case "ENTER", "RETURN":
		return "{ENTER}", nil
	case "TAB":
		return "{TAB}", nil
	case "ESC", "ESCAPE":
		return "{ESC}", nil
	case "BACKSPACE":
		return "{BACKSPACE}", nil
	case "DELETE", "DEL":
		return "{DELETE}", nil
	case "SPACE":
		return " ", nil
	case "UP":
		return "{UP}", nil
	case "DOWN":
		return "{DOWN}", nil
	case "LEFT":
		return "{LEFT}", nil
	case "RIGHT":
		return "{RIGHT}", nil
	default:
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			return "", errors.New("key is empty")
		}
		if len(trimmed) == 1 {
			return toSendKeysLiteral(trimmed), nil
		}
		if strings.HasPrefix(strings.ToUpper(trimmed), "F") {
			return "{" + strings.ToUpper(trimmed) + "}", nil
		}
		return "", fmt.Errorf("unsupported key: %s", key)
	}
}

func toHotkeySendKeys(keys []string) (string, error) {
	if len(keys) == 0 {
		return "", errors.New("keys are empty")
	}
	modifiers := make([]string, 0, len(keys))
	regular := make([]string, 0, len(keys))
	for _, key := range keys {
		switch strings.ToUpper(strings.TrimSpace(key)) {
		case "CTRL", "CONTROL":
			modifiers = append(modifiers, "^")
		case "ALT":
			modifiers = append(modifiers, "%")
		case "SHIFT":
			modifiers = append(modifiers, "+")
		default:
			token, err := toSendKeysToken(key)
			if err != nil {
				return "", err
			}
			regular = append(regular, token)
		}
	}
	if len(regular) == 0 {
		return "", errors.New("hotkey requires at least one non-modifier key")
	}
	if len(regular) == 1 {
		return strings.Join(modifiers, "") + regular[0], nil
	}
	return strings.Join(modifiers, "") + "(" + strings.Join(regular, "") + ")", nil
}
