package node

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestBuildWSLCommandScriptExportsSortedEnv(t *testing.T) {
	script, err := buildWSLCommandScript("git", []string{"status"}, map[string]string{
		"Z_VAR": "2",
		"A_VAR": "1",
	})
	if err != nil {
		t.Fatalf("build command script: %v", err)
	}

	if !strings.Contains(script, "set -euo pipefail") {
		t.Fatalf("expected strict shell flags, got %q", script)
	}
	if !strings.Contains(script, "export A_VAR='1'") || !strings.Contains(script, "export Z_VAR='2'") {
		t.Fatalf("expected exported env vars, got %q", script)
	}
	if strings.Index(script, "export A_VAR='1'") > strings.Index(script, "export Z_VAR='2'") {
		t.Fatalf("expected environment variables to be sorted, got %q", script)
	}
	if !strings.HasSuffix(script, "exec 'git' 'status'") {
		t.Fatalf("expected command to be appended at the end, got %q", script)
	}
}

func TestParseWSLListOutput(t *testing.T) {
	raw := []byte("README.md\x00f\x0012\x001710000000.5\x00src\x00d\x000\x001710000001.0\x00")
	items, err := parseWSLListOutput(raw)
	if err != nil {
		t.Fatalf("parse list output: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 entries, got %+v", items)
	}

	if items[0]["name"] != "README.md" || items[0]["is_dir"] != false || items[0]["size"] != int64(12) {
		t.Fatalf("unexpected first entry: %+v", items[0])
	}
	if items[1]["name"] != "src" || items[1]["is_dir"] != true {
		t.Fatalf("unexpected second entry: %+v", items[1])
	}
}

func TestDecodeWSLTextUTF16LE(t *testing.T) {
	text := "Ubuntu-22.04\r\n"
	units := utf16.Encode([]rune(text))
	raw := make([]byte, len(units)*2)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(raw[index*2:], unit)
	}

	if got := decodeWSLText(raw); got != text {
		t.Fatalf("unexpected decoded text: got %q want %q", got, text)
	}
}

func TestWSLDriverExecIntegration(t *testing.T) {
	if !isWSLRuntime() {
		t.Skip("requires WSL runtime")
	}
	if os.Getenv("TINYCLAW_RUN_WSL_INTEGRATION") != "1" {
		t.Skip("set TINYCLAW_RUN_WSL_INTEGRATION=1 to run WSL integration checks")
	}

	distro := integrationTestWSLDistro(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	driver := NewWSLDriver(WSLDriverConfig{DistroName: distro, DefaultCWD: cwd})
	result, err := driver.Execute(context.Background(), NodeCommandRequest{
		ID:         "wsl-exec-integration",
		NodeID:     "node-wsl",
		Capability: "wsl.exec",
		Arguments: map[string]interface{}{
			"command": "bash",
			"args":    []interface{}{"-lc", "pwd && uname -s && git status --short --branch"},
		},
	})
	if err != nil {
		t.Fatalf("execute integration command: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if !strings.Contains(result.Output, cwd) || !strings.Contains(result.Output, "Linux") || !strings.Contains(result.Output, "##") {
		t.Fatalf("unexpected command output: %q", result.Output)
	}
}

func TestWSLDriverFileRoundTripIntegration(t *testing.T) {
	if !isWSLRuntime() {
		t.Skip("requires WSL runtime")
	}
	if os.Getenv("TINYCLAW_RUN_WSL_INTEGRATION") != "1" {
		t.Skip("set TINYCLAW_RUN_WSL_INTEGRATION=1 to run WSL integration checks")
	}

	distro := integrationTestWSLDistro(t)
	driver := NewWSLDriver(WSLDriverConfig{DistroName: distro})

	targetDir := t.TempDir()
	targetFile := filepath.Join(targetDir, "tinyclaw-wsl.txt")

	if _, err := driver.Execute(context.Background(), NodeCommandRequest{
		ID:         "wsl-write-integration",
		NodeID:     "node-wsl",
		Capability: "wsl.fs.write",
		Arguments: map[string]interface{}{
			"path":    targetFile,
			"content": "hello from wsl",
		},
	}); err != nil {
		t.Fatalf("write integration file: %v", err)
	}

	readResult, err := driver.Execute(context.Background(), NodeCommandRequest{
		ID:         "wsl-read-integration",
		NodeID:     "node-wsl",
		Capability: "wsl.fs.read",
		Arguments: map[string]interface{}{
			"path": targetFile,
		},
	})
	if err != nil {
		t.Fatalf("read integration file: %v", err)
	}
	if readResult.Output != "hello from wsl" {
		t.Fatalf("unexpected read output: %q", readResult.Output)
	}

	listResult, err := driver.Execute(context.Background(), NodeCommandRequest{
		ID:         "wsl-list-integration",
		NodeID:     "node-wsl",
		Capability: "wsl.fs.list",
		Arguments: map[string]interface{}{
			"path": targetDir,
		},
	})
	if err != nil {
		t.Fatalf("list integration dir: %v", err)
	}

	entries, _ := listResult.Data["entries"].([]map[string]interface{})
	if len(entries) == 0 {
		raw, ok := listResult.Data["entries"].([]interface{})
		if ok {
			for _, item := range raw {
				entry, ok := item.(map[string]interface{})
				if ok && entry["name"] == "tinyclaw-wsl.txt" {
					return
				}
			}
		}
		t.Fatalf("expected integration directory listing to include tinyclaw-wsl.txt, got %+v", listResult.Data["entries"])
	}
	found := false
	for _, entry := range entries {
		if entry["name"] == "tinyclaw-wsl.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected integration directory listing to include tinyclaw-wsl.txt, got %+v", entries)
	}
}

func integrationTestWSLDistro(t *testing.T) string {
	t.Helper()
	if distro := strings.TrimSpace(os.Getenv("TINYCLAW_WSL_DISTRO")); distro != "" {
		return distro
	}

	distros, err := ListWSLDistros(context.Background())
	if err != nil {
		t.Fatalf("list WSL distros: %v", err)
	}
	for _, distro := range distros {
		if trimmed := strings.TrimSpace(distro); trimmed != "" && !strings.Contains(strings.ToLower(trimmed), "docker-desktop") {
			return trimmed
		}
	}
	if len(distros) == 0 {
		t.Fatal("no WSL distros available for integration testing")
	}
	return strings.TrimSpace(distros[0])
}
