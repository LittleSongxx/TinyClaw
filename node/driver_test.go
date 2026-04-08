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

func TestStringArgHandlesNumericHandle(t *testing.T) {
	args := map[string]interface{}{
		"window_handle": float64(123456),
	}
	if got := stringArg(args, "window_handle"); got != "123456" {
		t.Fatalf("expected numeric handle to be converted, got %q", got)
	}
}

func TestLocalDriverExecuteReturnsErrorResultWithoutPanic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this regression only manifests on non-windows fallback paths")
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
