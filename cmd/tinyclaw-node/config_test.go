package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveProcessConfigPrecedence(t *testing.T) {
	t.Setenv("NODE_GATEWAY_WS", "ws://env.example/gateway")
	t.Setenv("NODE_PAIRING_TOKEN", "env-token")

	configPath := filepath.Join(t.TempDir(), "config.json")
	content := `{
  "gateway_ws": "ws://file.example/gateway",
  "node_token": "file-token",
  "node_id": "file-node-id",
  "node_name": "file-node-name",
  "log_dir": "/tmp/tinyclaw-file-log",
  "enable_windows_node": false
}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg, err := resolveProcessConfig(ctx, cliOptions{
		ConfigPath: configPath,
		GatewayWS:  "ws://cli.example/gateway",
		NodeName:   "cli-node-name",
		NodeToken:  "cli-token",
	}, map[string]bool{
		"gateway_ws": true,
		"node_name":  true,
		"node_token": true,
	})
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}

	if cfg.GatewayWS != "ws://cli.example/gateway" {
		t.Fatalf("expected CLI gateway to win, got %q", cfg.GatewayWS)
	}
	if cfg.NodeToken != "cli-token" {
		t.Fatalf("expected CLI token to win, got %q", cfg.NodeToken)
	}
	if cfg.NodeID != "file-node-id" {
		t.Fatalf("expected file node id, got %q", cfg.NodeID)
	}
	if cfg.NodeName != "cli-node-name" {
		t.Fatalf("expected CLI node name to win, got %q", cfg.NodeName)
	}
	if cfg.LogDir != "/tmp/tinyclaw-file-log" {
		t.Fatalf("expected file log dir, got %q", cfg.LogDir)
	}
	if cfg.EnableWindowsNode {
		t.Fatal("expected file config to disable the Windows node")
	}
}

func TestNormalizeProcessConfigNormalizesAllowlists(t *testing.T) {
	cfg := processConfig{
		GatewayWS: "ws://127.0.0.1:36060/gateway/nodes/ws",
		NodeToken: "token",
		NodeID:    "node-1",
		NodeName:  "node-1",
		WSLDistros: []wslDistroConfig{
			{
				Name:                   "Ubuntu-22.04",
				AllowCommandPrefixes:   []string{"  git   status  ", "git status"},
				AllowWritePathPrefixes: []string{" /workspace/project/../project ", "/workspace/project"},
			},
		},
	}

	normalizeProcessConfig(&cfg)

	if got, want := cfg.WSLDistros[0].AllowCommandPrefixes, []string{"git status"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected command allowlist: got %v want %v", got, want)
	}
	if got, want := cfg.WSLDistros[0].AllowWritePathPrefixes, []string{"/workspace/project"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected path allowlist: got %v want %v", got, want)
	}
}

func TestLoadProcessConfigAcceptsUTF8BOM(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	content := []byte(`{"gateway_ws":"ws://127.0.0.1:36060/gateway/nodes/ws","node_token":"token"}`)
	content = append([]byte{0xEF, 0xBB, 0xBF}, content...)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadProcessConfig(configPath)
	if err != nil {
		t.Fatalf("load config with BOM: %v", err)
	}
	if cfg.GatewayWS == nil || *cfg.GatewayWS != "ws://127.0.0.1:36060/gateway/nodes/ws" {
		t.Fatalf("unexpected gateway_ws: %+v", cfg.GatewayWS)
	}
	if cfg.NodeToken == nil || *cfg.NodeToken != "token" {
		t.Fatalf("unexpected node_token: %+v", cfg.NodeToken)
	}
}
