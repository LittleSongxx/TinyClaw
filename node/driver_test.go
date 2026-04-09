package node

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLocalDriverFileRoundTrip(t *testing.T) {
	driver := NewLocalDriver()
	ctx := context.Background()
	target := filepath.Join(t.TempDir(), "note.txt")

	if _, err := driver.Execute(ctx, NodeCommandRequest{
		ID:         "write-1",
		Capability: "fs.write",
		Arguments: map[string]interface{}{
			"path":    target,
			"content": "hello tinyclaw",
		},
	}); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result, err := driver.Execute(ctx, NodeCommandRequest{
		ID:         "read-1",
		Capability: "fs.read",
		Arguments: map[string]interface{}{
			"path": target,
		},
	})
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if result.Output != "hello tinyclaw" {
		t.Fatalf("unexpected read output: %q", result.Output)
	}
}

func TestLocalDriverExecCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("system.exec test currently uses sh")
	}
	driver := NewLocalDriver()
	ctx := context.Background()

	result, err := driver.Execute(ctx, NodeCommandRequest{
		ID:         "exec-1",
		Capability: "system.exec",
		Arguments: map[string]interface{}{
			"command": "sh",
			"args":    []interface{}{"-c", "printf tinyclaw"},
		},
	})
	if err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Output != "tinyclaw" {
		t.Fatalf("unexpected command output: %q", result.Output)
	}
}

func TestInferScreenshotMeta(t *testing.T) {
	target := filepath.Join(t.TempDir(), "snapshot.png")
	img := image.NewRGBA(image.Rect(0, 0, 320, 180))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	file, err := os.Create(target)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatalf("encode png: %v", err)
	}
	file.Close()

	meta, err := inferScreenshotMeta(target, "virtual_desktop")
	if err != nil {
		t.Fatalf("infer screenshot meta: %v", err)
	}
	if meta.Scope != "virtual_desktop" || meta.Width != 320 || meta.Height != 180 {
		t.Fatalf("unexpected screenshot meta: %+v", meta)
	}
}

func TestToHotkeySendKeys(t *testing.T) {
	got, err := toHotkeySendKeys([]string{"CTRL", "SHIFT", "S"})
	if err != nil {
		t.Fatalf("toHotkeySendKeys: %v", err)
	}
	if got != "^+S" {
		t.Fatalf("unexpected hotkey sequence: %s", got)
	}
}

func TestNodeCapabilitiesMatchDesktopAutomationSupport(t *testing.T) {
	withDesktop := nodeCapabilitiesForRuntime()
	names := make(map[string]bool, len(withDesktop))
	for _, capability := range withDesktop {
		names[capability.Name] = true
	}

	if !names["screen.snapshot"] || !names["system.exec"] {
		t.Fatalf("expected base capabilities to always be registered, got %+v", withDesktop)
	}

	if supportsWindowsDesktopAutomation() {
		for _, capability := range []string{"input.keyboard.type", "input.mouse.click", "window.list", "ui.find"} {
			if !names[capability] {
				t.Fatalf("expected desktop capability %s in %+v", capability, withDesktop)
			}
		}
		return
	}

	for _, capability := range []string{"input.keyboard.type", "input.mouse.click", "window.list", "ui.find"} {
		if names[capability] {
			t.Fatalf("did not expect desktop capability %s in %+v", capability, withDesktop)
		}
	}
}

func TestWSLDesktopScreenshotBridge(t *testing.T) {
	if !isWSLRuntime() {
		t.Skip("requires WSL")
	}
	if os.Getenv("TINYCLAW_RUN_WSL_INTEGRATION") != "1" {
		t.Skip("set TINYCLAW_RUN_WSL_INTEGRATION=1 to run WSL integration checks")
	}

	target, captureTarget, err := desktopTempFilePath("tinyclaw-wsl-screenshot-test.png")
	if err != nil {
		t.Fatalf("resolve screenshot target: %v", err)
	}
	defer os.Remove(target)

	meta, err := captureScreenshot(context.Background(), target, captureTarget, "primary", map[string]interface{}{
		"scope": "primary",
	})
	if err != nil {
		t.Fatalf("capture windows screenshot from wsl: %v", err)
	}
	if meta == nil || meta.Width <= 0 || meta.Height <= 0 {
		t.Fatalf("unexpected screenshot meta: %+v", meta)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected screenshot file at %s: %v", target, err)
	}
}

func TestStringArgHandlesNumericHandle(t *testing.T) {
	args := map[string]interface{}{
		"window_handle": float64(123456),
	}
	if got := stringArg(args, "window_handle"); got != "123456" {
		t.Fatalf("expected numeric handle to be converted, got %q", got)
	}
}

func TestLocalDriverExecuteReturnsErrorResultWithoutPanic(t *testing.T) {
	if supportsWindowsDesktopAutomation() {
		t.Skip("desktop automation is available in this runtime")
	}
	driver := NewLocalDriver()
	result, err := driver.Execute(context.Background(), NodeCommandRequest{
		ID:         "keyboard-type-error",
		Capability: "input.keyboard.type",
		Arguments: map[string]interface{}{
			"text": "hello",
		},
	})
	if err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	if result == nil || result.Success || result.Error == "" {
		t.Fatalf("expected structured error result, got %+v", result)
	}
}
