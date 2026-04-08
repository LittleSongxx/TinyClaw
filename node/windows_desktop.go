package node

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

type windowBounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type windowMeta struct {
	Handle       int64        `json:"handle"`
	Title        string       `json:"title"`
	ProcessName  string       `json:"process_name"`
	ProcessID    int          `json:"process_id,omitempty"`
	IsForeground bool         `json:"is_foreground,omitempty"`
	Bounds       windowBounds `json:"bounds"`
}

func captureWindowsScreenshot(ctx context.Context, target string, args map[string]interface{}) (*screenshotMeta, error) {
	request := map[string]interface{}{
		"scope":         firstNonEmptyString(args, "scope"),
		"window_handle": firstNonEmptyString(args, "window_handle", "handle"),
		"window_title":  firstNonEmptyString(args, "window_title", "title"),
		"process_name":  firstNonEmptyString(args, "process_name"),
		"target":        target,
	}
	if request["scope"] == "" {
		request["scope"] = "virtual_desktop"
	}

	script := windowsDesktopAutomationPrelude() + `
$request = '` + marshalPowerShellJSON(request) + `' | ConvertFrom-Json
$scope = [string]$request.scope
if (-not $scope) { $scope = "virtual_desktop" }
$windowInfo = $null
if ($scope -eq "active_window") {
  $windowInfo = Resolve-TinyClawWindow $request
  if (-not $windowInfo) { throw "window not found" }
  $bounds = $windowInfo.bounds
} elseif ($scope -eq "primary") {
  $bounds = [ordered]@{
    x = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds.X
    y = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds.Y
    width = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds.Width
    height = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds.Height
  }
} elseif ($scope -eq "virtual_desktop") {
  $virtual = [System.Windows.Forms.SystemInformation]::VirtualScreen
  $bounds = [ordered]@{
    x = $virtual.X
    y = $virtual.Y
    width = $virtual.Width
    height = $virtual.Height
  }
} else {
  throw ("unsupported screenshot scope: " + $scope)
}
if (-not $bounds -or $bounds.width -le 0 -or $bounds.height -le 0) {
  throw "invalid screenshot bounds"
}
$bitmap = New-Object System.Drawing.Bitmap ([int]$bounds.width), ([int]$bounds.height)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.CopyFromScreen([int]$bounds.x, [int]$bounds.y, 0, 0, $bitmap.Size)
$bitmap.Save([string]$request.target, [System.Drawing.Imaging.ImageFormat]::Png)
$graphics.Dispose()
$bitmap.Dispose()
$payload = [ordered]@{
  scope = $scope
  width = [int]$bounds.width
  height = [int]$bounds.height
  display_count = [System.Windows.Forms.Screen]::AllScreens.Length
}
if ($windowInfo) {
  $payload.window = $windowInfo
}
$payload | ConvertTo-Json -Compress -Depth 16
`

	raw, err := runPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	meta := &screenshotMeta{}
	if err := decodePowerShellJSON(raw, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func listWindowsDetailed(ctx context.Context) ([]map[string]interface{}, error) {
	script := windowsDesktopAutomationPrelude() + `
$foreground = [TinyClawUser32]::GetForegroundWindow().ToInt64()
$windows = Get-Process |
  Where-Object { $_.MainWindowHandle -ne 0 -and $_.MainWindowTitle } |
  Sort-Object MainWindowTitle |
  ForEach-Object {
    $info = Get-TinyClawWindowInfo $_.MainWindowHandle
    if ($info) {
      if ([int64]$info.handle -eq $foreground) { $info.is_foreground = $true }
      $info
    }
  }
$windows | ConvertTo-Json -Compress -Depth 16
`
	raw, err := runPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	return decodePowerShellJSONArray(raw)
}

func focusWindowDetailed(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	request := map[string]interface{}{
		"window_handle": firstNonEmptyString(args, "window_handle", "handle"),
		"window_title":  firstNonEmptyString(args, "window_title", "title"),
		"process_name":  firstNonEmptyString(args, "process_name"),
	}
	script := windowsDesktopAutomationPrelude() + `
$request = '` + marshalPowerShellJSON(request) + `' | ConvertFrom-Json
$target = Resolve-TinyClawWindow $request
if (-not $target) { throw "window not found" }
[TinyClawUser32]::ShowWindowAsync([IntPtr]::new([int64]$target.handle), 5) | Out-Null
[TinyClawUser32]::SetForegroundWindow([IntPtr]::new([int64]$target.handle)) | Out-Null
$target.is_foreground = $true
$target | ConvertTo-Json -Compress -Depth 16
`
	raw, err := runPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	return decodePowerShellJSONObject(raw)
}

func inspectWindowsUI(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	request := map[string]interface{}{
		"mode":          firstNonEmptyString(args, "mode"),
		"depth":         intArg(args, "depth", 4),
		"x":             numericArg(args, "x"),
		"y":             numericArg(args, "y"),
		"window_handle": firstNonEmptyString(args, "window_handle", "handle"),
		"window_title":  firstNonEmptyString(args, "window_title", "title"),
		"process_name":  firstNonEmptyString(args, "process_name"),
	}
	if request["mode"] == "" {
		request["mode"] = "window_tree"
	}

	script := windowsDesktopAutomationPrelude() + `
$request = '` + marshalPowerShellJSON(request) + `' | ConvertFrom-Json
$mode = [string]$request.mode
if (-not $mode) { $mode = "window_tree" }
$depth = [int]$request.depth
if ($depth -le 0) { $depth = 4 }

if ($mode -eq "window_tree") {
  $windowInfo = Resolve-TinyClawWindow $request
  if (-not $windowInfo) { throw "window not found" }
  $rootElement = Get-TinyClawWindowElement $windowInfo
  if (-not $rootElement) { throw "window ui root not found" }
  $rootNode = Get-TinyClawAutomationNode $rootElement "0" 0 $depth
  $nodes = @()
  if ($rootNode) { $nodes += $rootNode }
  [ordered]@{
    mode = $mode
    window = $windowInfo
    focused_path = Get-TinyClawFocusedPath $nodes
    nodes = $nodes
  } | ConvertTo-Json -Compress -Depth 32
  return
}

if ($mode -eq "point") {
  if ($null -eq $request.x -or $null -eq $request.y) { throw "ui.inspect point mode requires x and y" }
  $point = New-Object System.Windows.Point([int]$request.x, [int]$request.y)
  $element = [System.Windows.Automation.AutomationElement]::FromPoint($point)
} else {
  $element = [System.Windows.Automation.AutomationElement]::FocusedElement
}
if (-not $element) { throw "ui element not found" }
$windowInfo = Get-TinyClawWindowInfoFromElement $element
$node = Get-TinyClawAutomationNodeSummary $element ""
[ordered]@{
  mode = $mode
  window = $windowInfo
  element = $node
  focused_path = ""
  nodes = @()
} | ConvertTo-Json -Compress -Depth 24
`

	raw, err := runPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	return decodePowerShellJSONObject(raw)
}

func findWindowsUI(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	request := map[string]interface{}{
		"window_handle": firstNonEmptyString(args, "window_handle", "handle"),
		"window_title":  firstNonEmptyString(args, "window_title", "title"),
		"process_name":  firstNonEmptyString(args, "process_name"),
		"automation_id": firstNonEmptyString(args, "automation_id"),
		"name":          firstNonEmptyString(args, "name"),
		"role":          firstNonEmptyString(args, "role"),
		"class_name":    firstNonEmptyString(args, "class_name"),
		"path":          firstNonEmptyString(args, "path"),
		"index":         intArg(args, "index", 0),
		"exact":         boolArg(args, "exact"),
		"max_results":   intArg(args, "max_results", 10),
		"depth":         intArg(args, "depth", 6),
	}
	script := windowsDesktopAutomationPrelude() + `
$request = '` + marshalPowerShellJSON(request) + `' | ConvertFrom-Json
$depth = [int]$request.depth
if ($depth -le 0) { $depth = 6 }
$maxResults = [int]$request.max_results
if ($maxResults -le 0) { $maxResults = 10 }
$windowInfo = Resolve-TinyClawWindow $request
if (-not $windowInfo) { throw "window not found" }
$rootElement = Get-TinyClawWindowElement $windowInfo
if (-not $rootElement) { throw "window ui root not found" }
$matches = Find-TinyClawElementMatches $rootElement $request $depth
$resultNodes = @()
foreach ($match in $matches) {
  $node = Add-TinyClawLocatorWindowData $match.node $windowInfo
  $resultNodes += $node
  if ($resultNodes.Count -ge $maxResults) { break }
}
[ordered]@{
  window = $windowInfo
  matches = $resultNodes
  count = $matches.Count
} | ConvertTo-Json -Compress -Depth 24
`
	raw, err := runPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	return decodePowerShellJSONObject(raw)
}

func typeIntoWindowsElement(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	text := stringArg(args, "text")
	if text == "" {
		return nil, errors.New("input.keyboard.type requires text")
	}
	element := elementArg(args)
	if len(element) == 0 {
		return nil, nil
	}

	request := map[string]interface{}{
		"text":    text,
		"element": element,
	}
	script := windowsDesktopAutomationPrelude() + `
$request = '` + marshalPowerShellJSON(request) + `' | ConvertFrom-Json
$resolved = Resolve-TinyClawElementContext $request.element 8
if (-not $resolved.element) { throw "ui element not found" }
$element = $resolved.element
$node = Add-TinyClawLocatorWindowData $resolved.node $resolved.window
$usedValuePattern = $false
$patternObject = $null
if ($element.TryGetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern, [ref]$patternObject)) {
  $pattern = [System.Windows.Automation.ValuePattern]$patternObject
  $pattern.SetValue([string]$request.text)
  $usedValuePattern = $true
} else {
  $element.SetFocus()
  Start-Sleep -Milliseconds 80
  [System.Windows.Forms.SendKeys]::SendWait('` + escapePowerShellSingleQuoted(toSendKeysLiteral(text)) + `')
}
[ordered]@{
  text = [string]$request.text
  method = $(if ($usedValuePattern) { "value_pattern" } else { "send_keys" })
  element = $node
} | ConvertTo-Json -Compress -Depth 16
`
	raw, err := runPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	return decodePowerShellJSONObject(raw)
}

func focusWindowsElement(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	element := elementArg(args)
	if len(element) == 0 {
		return nil, nil
	}
	request := map[string]interface{}{
		"element": element,
	}
	script := windowsDesktopAutomationPrelude() + `
$request = '` + marshalPowerShellJSON(request) + `' | ConvertFrom-Json
$resolved = Resolve-TinyClawElementContext $request.element 8
if (-not $resolved.element) { throw "ui element not found" }
$resolved.element.SetFocus()
Add-TinyClawLocatorWindowData $resolved.node $resolved.window | ConvertTo-Json -Compress -Depth 16
`
	raw, err := runPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	return decodePowerShellJSONObject(raw)
}

func clickWindowsElement(ctx context.Context, args map[string]interface{}, button string, clickCount int) (map[string]interface{}, error) {
	element := elementArg(args)
	if len(element) == 0 {
		return nil, nil
	}
	if button == "" {
		button = "left"
	}
	request := map[string]interface{}{
		"element": element,
		"button":  button,
		"clicks":  clickCount,
	}
	script := windowsDesktopAutomationPrelude() + `
$request = '` + marshalPowerShellJSON(request) + `' | ConvertFrom-Json
$resolved = Resolve-TinyClawElementContext $request.element 8
if (-not $resolved.element) { throw "ui element not found" }
$node = Add-TinyClawLocatorWindowData $resolved.node $resolved.window
$bounds = $node.bounds
if (-not $bounds -or $bounds.width -le 0 -or $bounds.height -le 0) { throw "element bounds are empty" }
$x = [int]($bounds.x + ($bounds.width / 2))
$y = [int]($bounds.y + ($bounds.height / 2))
[TinyClawUser32]::SetCursorPos($x, $y) | Out-Null
$button = [string]$request.button
$clicks = [int]$request.clicks
if ($clicks -le 0) { $clicks = 1 }
if ($button -eq "right") {
  $down = [TinyClawUser32]::MOUSEEVENTF_RIGHTDOWN
  $up = [TinyClawUser32]::MOUSEEVENTF_RIGHTUP
} else {
  $down = [TinyClawUser32]::MOUSEEVENTF_LEFTDOWN
  $up = [TinyClawUser32]::MOUSEEVENTF_LEFTUP
}
for ($i = 0; $i -lt $clicks; $i++) {
  [TinyClawUser32]::mouse_event($down, 0, 0, 0, [UIntPtr]::Zero)
  [TinyClawUser32]::mouse_event($up, 0, 0, 0, [UIntPtr]::Zero)
}
[ordered]@{
  button = $button
  clicks = $clicks
  x = $x
  y = $y
  element = $node
} | ConvertTo-Json -Compress -Depth 16
`
	raw, err := runPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	return decodePowerShellJSONObject(raw)
}

func elementArg(args map[string]interface{}) map[string]interface{} {
	if len(args) == 0 {
		return nil
	}
	raw, ok := args["element"]
	if !ok || raw == nil {
		return nil
	}
	switch value := raw.(type) {
	case map[string]interface{}:
		return cloneAnyMap(value)
	case map[string]string:
		out := make(map[string]interface{}, len(value))
		for key, item := range value {
			out[key] = item
		}
		return out
	default:
		return nil
	}
}

func cloneAnyMap(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func numericArg(args map[string]interface{}, key string) interface{} {
	if len(args) == 0 {
		return nil
	}
	if value, ok := args[key]; ok {
		switch typed := value.(type) {
		case int, int32, int64, float64:
			return typed
		}
	}
	return nil
}

func firstNonEmptyString(args map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value := stringArg(args, key)
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func marshalPowerShellJSON(value interface{}) string {
	content, _ := json.Marshal(value)
	return escapePowerShellSingleQuoted(string(content))
}

func windowsDesktopAutomationPrelude() string {
	return `
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type @"
using System;
using System.Runtime.InteropServices;
using System.Text;
public struct RECT {
  public int Left;
  public int Top;
  public int Right;
  public int Bottom;
}
public static class TinyClawUser32 {
  [DllImport("user32.dll")] public static extern bool SetCursorPos(int X, int Y);
  [DllImport("user32.dll")] public static extern void mouse_event(uint dwFlags, uint dx, uint dy, uint dwData, UIntPtr dwExtraInfo);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow);
  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);
  [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId);
  public const uint MOUSEEVENTF_LEFTDOWN = 0x0002;
  public const uint MOUSEEVENTF_LEFTUP = 0x0004;
  public const uint MOUSEEVENTF_RIGHTDOWN = 0x0008;
  public const uint MOUSEEVENTF_RIGHTUP = 0x0010;
}
public static class TinyClawDwmApi {
  [DllImport("dwmapi.dll")] public static extern int DwmGetWindowAttribute(IntPtr hwnd, int attribute, out RECT rect, int cbAttribute);
  public const int DWMWA_EXTENDED_FRAME_BOUNDS = 9;
}
"@

function Convert-TinyClawRect($rect) {
  if ($null -eq $rect) { return $null }
  [ordered]@{
    x = [int]$rect.X
    y = [int]$rect.Y
    width = [int]$rect.Width
    height = [int]$rect.Height
  }
}

function Convert-TinyClawNativeRect($rect) {
  [ordered]@{
    x = [int]$rect.Left
    y = [int]$rect.Top
    width = [int]($rect.Right - $rect.Left)
    height = [int]($rect.Bottom - $rect.Top)
  }
}

function Get-TinyClawWindowBounds([IntPtr]$handle) {
  $rect = New-Object RECT
  $result = [TinyClawDwmApi]::DwmGetWindowAttribute($handle, [TinyClawDwmApi]::DWMWA_EXTENDED_FRAME_BOUNDS, [ref]$rect, [Runtime.InteropServices.Marshal]::SizeOf([type][RECT]))
  if ($result -eq 0) {
    return Convert-TinyClawNativeRect $rect
  }
  if ([TinyClawUser32]::GetWindowRect($handle, [ref]$rect)) {
    return Convert-TinyClawNativeRect $rect
  }
  return $null
}

function Get-TinyClawWindowInfo([IntPtr]$handle) {
  if ($handle -eq [IntPtr]::Zero) { return $null }
  [uint32]$processId = 0
  [TinyClawUser32]::GetWindowThreadProcessId($handle, [ref]$processId) | Out-Null
  $process = $null
  if ($processId -gt 0) {
    $process = Get-Process -Id $processId -ErrorAction SilentlyContinue | Select-Object -First 1
  }
  $bounds = Get-TinyClawWindowBounds $handle
  [ordered]@{
    handle = [int64]$handle
    title = $(if ($process) { $process.MainWindowTitle } else { "" })
    process_name = $(if ($process) { $process.ProcessName } else { "" })
    process_id = [int]$processId
    is_foreground = ([TinyClawUser32]::GetForegroundWindow().ToInt64() -eq $handle.ToInt64())
    bounds = $bounds
  }
}

function Resolve-TinyClawWindow($selector) {
  if ($null -eq $selector) { $selector = @{} }

  $handleValue = $null
  if ($selector.PSObject.Properties["window_handle"]) { $handleValue = $selector.window_handle }
  if (($null -eq $handleValue -or "$handleValue" -eq "") -and $selector.PSObject.Properties["handle"]) { $handleValue = $selector.handle }
  if ($null -ne $handleValue -and "$handleValue" -ne "") {
    return Get-TinyClawWindowInfo([IntPtr]::new([int64]$handleValue))
  }

  $title = ""
  if ($selector.PSObject.Properties["window_title"] -and $selector.window_title) { $title = [string]$selector.window_title }
  if (-not $title -and $selector.PSObject.Properties["title"] -and $selector.title) { $title = [string]$selector.title }
  $processName = ""
  if ($selector.PSObject.Properties["process_name"] -and $selector.process_name) { $processName = [string]$selector.process_name }

  if (-not $title -and -not $processName) {
    return Get-TinyClawWindowInfo([TinyClawUser32]::GetForegroundWindow())
  }

  $windows = Get-Process |
    Where-Object { $_.MainWindowHandle -ne 0 -and $_.MainWindowTitle }

  if ($processName) {
    $windows = $windows | Where-Object { $_.ProcessName -eq $processName -or $_.ProcessName -like ("*" + $processName + "*") }
  }
  if ($title) {
    $exact = $windows | Where-Object { $_.MainWindowTitle -eq $title } | Select-Object -First 1
    if ($exact) { return Get-TinyClawWindowInfo $exact.MainWindowHandle }
    $windows = $windows | Where-Object { $_.MainWindowTitle -like ("*" + $title + "*") }
  }

  $target = $windows | Select-Object -First 1
  if (-not $target) { return $null }
  return Get-TinyClawWindowInfo $target.MainWindowHandle
}

function Get-TinyClawWindowElement($windowInfo) {
  if ($null -eq $windowInfo -or $null -eq $windowInfo.handle) { return $null }
  return [System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]::new([int64]$windowInfo.handle))
}

function Get-TinyClawControlType($element) {
  try {
    return $element.Current.ControlType.ProgrammaticName
  } catch {
    return ""
  }
}

function Get-TinyClawLocalizedControlType($element) {
  try {
    return $element.Current.LocalizedControlType
  } catch {
    return ""
  }
}

function Get-TinyClawAutomationNodeSummary($element, [string]$path) {
  if ($null -eq $element) { return $null }
  try {
    $rect = $element.Current.BoundingRectangle
    return [ordered]@{
      path = $path
      name = $element.Current.Name
      automation_id = $element.Current.AutomationId
      role = Get-TinyClawControlType $element
      control_type = Get-TinyClawLocalizedControlType $element
      class_name = $element.Current.ClassName
      bounds = Convert-TinyClawRect $rect
      is_enabled = [bool]$element.Current.IsEnabled
      is_offscreen = [bool]$element.Current.IsOffscreen
      has_keyboard_focus = [bool]$element.Current.HasKeyboardFocus
      children = @()
    }
  } catch {
    return $null
  }
}

function Get-TinyClawAutomationNode($element, [string]$path, [int]$depth, [int]$maxDepth) {
  $node = Get-TinyClawAutomationNodeSummary $element $path
  if ($null -eq $node) { return $null }
  if ($depth -ge $maxDepth) {
    return $node
  }
  $walker = [System.Windows.Automation.TreeWalker]::ControlViewWalker
  $child = $walker.GetFirstChild($element)
  $index = 0
  while ($child -ne $null) {
    $childNode = Get-TinyClawAutomationNode $child ($path + "." + $index) ($depth + 1) $maxDepth
    if ($childNode) {
      $node.children += $childNode
    }
    $child = $walker.GetNextSibling($child)
    $index++
  }
  return $node
}

function Get-TinyClawFocusedPath($nodes) {
  foreach ($node in $nodes) {
    if ($node.has_keyboard_focus) { return $node.path }
    if ($node.children -and $node.children.Count -gt 0) {
      $childPath = Get-TinyClawFocusedPath $node.children
      if ($childPath) { return $childPath }
    }
  }
  return ""
}

function Get-TinyClawTopWindowElement($element) {
  if ($null -eq $element) { return $null }
  $walker = [System.Windows.Automation.TreeWalker]::RawViewWalker
  $current = $element
  $candidate = $null
  while ($current -ne $null) {
    try {
      $handle = [int64]$current.Current.NativeWindowHandle
      if ($handle -ne 0) {
        $candidate = $current
      }
    } catch {}
    $parent = $walker.GetParent($current)
    if ($parent -eq $null) { break }
    $current = $parent
  }
  return $candidate
}

function Get-TinyClawWindowInfoFromElement($element) {
  $windowElement = Get-TinyClawTopWindowElement $element
  if ($windowElement) {
    try {
      $handle = [int64]$windowElement.Current.NativeWindowHandle
      if ($handle -ne 0) {
        return Get-TinyClawWindowInfo([IntPtr]::new($handle))
      }
    } catch {}
  }
  return Get-TinyClawWindowInfo([TinyClawUser32]::GetForegroundWindow())
}

function Get-TinyClawElementByPath($root, [string]$path) {
  if (-not $path) { return $root }
  $parts = $path -split "\."
  $current = $root
  for ($i = 1; $i -lt $parts.Length; $i++) {
    $index = [int]$parts[$i]
    $walker = [System.Windows.Automation.TreeWalker]::ControlViewWalker
    $child = $walker.GetFirstChild($current)
    $cursor = 0
    while ($child -ne $null -and $cursor -lt $index) {
      $child = $walker.GetNextSibling($child)
      $cursor++
    }
    if ($child -eq $null) { return $null }
    $current = $child
  }
  return $current
}

function Test-TinyClawStringMatch($candidate, $expected, [bool]$exact) {
  $candidateValue = ""
  if ($null -ne $candidate) { $candidateValue = [string]$candidate }
  $expectedValue = ""
  if ($null -ne $expected) { $expectedValue = [string]$expected }
  if (-not $expectedValue) { return $true }
  if (-not $candidateValue) { return $false }
  if ($exact) {
    return $candidateValue.Equals($expectedValue, [System.StringComparison]::OrdinalIgnoreCase)
  }
  return $candidateValue.IndexOf($expectedValue, [System.StringComparison]::OrdinalIgnoreCase) -ge 0
}

function Test-TinyClawNodeMatch($node, $locator) {
  $hasCriteria = $false
  if ($locator.PSObject.Properties["path"] -and $locator.path) {
    return ([string]$node.path -eq [string]$locator.path)
  }
  $exact = $false
  if ($locator.PSObject.Properties["exact"]) { $exact = [bool]$locator.exact }
  if ($locator.PSObject.Properties["automation_id"] -and $locator.automation_id) {
    $hasCriteria = $true
    if (-not (Test-TinyClawStringMatch $node.automation_id $locator.automation_id $exact)) { return $false }
  }
  if ($locator.PSObject.Properties["name"] -and $locator.name) {
    $hasCriteria = $true
    if (-not (Test-TinyClawStringMatch $node.name $locator.name $exact)) { return $false }
  }
  if ($locator.PSObject.Properties["role"] -and $locator.role) {
    $hasCriteria = $true
    if (-not (Test-TinyClawStringMatch $node.role $locator.role $exact) -and -not (Test-TinyClawStringMatch $node.control_type $locator.role $exact)) {
      return $false
    }
  }
  if ($locator.PSObject.Properties["class_name"] -and $locator.class_name) {
    $hasCriteria = $true
    if (-not (Test-TinyClawStringMatch $node.class_name $locator.class_name $exact)) { return $false }
  }
  return $hasCriteria
}

function Find-TinyClawElementMatchesRecursive($element, [string]$path, [int]$depth, [int]$maxDepth, $locator, [System.Collections.ArrayList]$matches) {
  $node = Get-TinyClawAutomationNodeSummary $element $path
  if ($null -eq $node) { return }
  if (Test-TinyClawNodeMatch $node $locator) {
    $matches.Add([ordered]@{ node = $node; element = $element }) | Out-Null
  }
  if ($depth -ge $maxDepth) { return }
  $walker = [System.Windows.Automation.TreeWalker]::ControlViewWalker
  $child = $walker.GetFirstChild($element)
  $index = 0
  while ($child -ne $null) {
    Find-TinyClawElementMatchesRecursive $child ($path + "." + $index) ($depth + 1) $maxDepth $locator $matches
    $child = $walker.GetNextSibling($child)
    $index++
  }
}

function Find-TinyClawElementMatches($rootElement, $locator, [int]$maxDepth) {
  $matches = New-Object System.Collections.ArrayList
  if ($locator.PSObject.Properties["path"] -and $locator.path) {
    $element = Get-TinyClawElementByPath $rootElement ([string]$locator.path)
    if ($element) {
      $matches.Add([ordered]@{
        node = Get-TinyClawAutomationNodeSummary $element ([string]$locator.path)
        element = $element
      }) | Out-Null
    }
    return $matches
  }
  Find-TinyClawElementMatchesRecursive $rootElement "0" 0 $maxDepth $locator $matches
  return $matches
}

function Add-TinyClawLocatorWindowData($node, $windowInfo) {
  if ($null -eq $node) { return $null }
  $copy = [ordered]@{}
  foreach ($property in $node.Keys) {
    $copy[$property] = $node[$property]
  }
  if ($windowInfo) {
    $copy.window_handle = $windowInfo.handle
    $copy.window_title = $windowInfo.title
    $copy.process_name = $windowInfo.process_name
  }
  return $copy
}

function Resolve-TinyClawElementContext($locator, [int]$maxDepth) {
  if ($null -eq $locator) { throw "element locator is required" }
  $windowSelector = [ordered]@{}
  if ($locator.PSObject.Properties["window_handle"] -and $locator.window_handle) { $windowSelector.window_handle = $locator.window_handle }
  if ($locator.PSObject.Properties["window_title"] -and $locator.window_title) { $windowSelector.window_title = $locator.window_title }
  if ($locator.PSObject.Properties["process_name"] -and $locator.process_name) { $windowSelector.process_name = $locator.process_name }
  $windowInfo = Resolve-TinyClawWindow $windowSelector
  if (-not $windowInfo) { $windowInfo = Resolve-TinyClawWindow @{} }
  if (-not $windowInfo) { throw "window not found" }
  $rootElement = Get-TinyClawWindowElement $windowInfo
  if (-not $rootElement) { throw "window ui root not found" }
  $matches = Find-TinyClawElementMatches $rootElement $locator $maxDepth
  if ($matches.Count -eq 0) {
    return [ordered]@{ window = $windowInfo; element = $null; node = $null; matches = @() }
  }
  $index = 0
  if ($locator.PSObject.Properties["index"] -and "$($locator.index)" -ne "") {
    $index = [int]$locator.index
  }
  if ($index -lt 0 -or $index -ge $matches.Count) { $index = 0 }
  $selected = $matches[$index]
  return [ordered]@{
    window = $windowInfo
    element = $selected.element
    node = $selected.node
    matches = $matches
  }
}
`
}
