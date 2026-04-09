package main

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/LittleSongxx/TinyClaw/node"
)

func TestBuildNodeInstancesFromDistrosIncludesWindowsAndWSLMetadata(t *testing.T) {
	cfg := processConfig{
		NodeID:            "desktop-node",
		NodeName:          "Desktop Node",
		EnableWindowsNode: true,
		WSLDistros: []wslDistroConfig{
			{
				Name:                   "Ubuntu-22.04",
				Enabled:                true,
				AllowCommandPrefixes:   []string{"git status"},
				AllowWritePathPrefixes: []string{"/workspace/project"},
				DefaultCWD:             "/workspace/project",
			},
		},
	}

	items, err := buildNodeInstancesFromDistros(cfg, "test-host", []string{"Ubuntu-22.04"})
	if err != nil {
		t.Fatalf("build node instances: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 node instances, got %d", len(items))
	}

	var windowsDesc *node.NodeDescriptor
	var wslDesc *node.NodeDescriptor
	for _, item := range items {
		switch item.descriptor.Platform {
		case runtime.GOOS:
			windowsDesc = item.descriptor
			if _, ok := item.driver.(*node.LocalDriver); !ok {
				t.Fatalf("expected Windows node to use LocalDriver, got %T", item.driver)
			}
		case "wsl":
			wslDesc = item.descriptor
			driver, ok := item.driver.(*node.WSLDriver)
			if !ok {
				t.Fatalf("expected WSL node to use WSLDriver, got %T", item.driver)
			}
			if driver == nil {
				t.Fatal("expected non-nil WSL driver")
			}
		}
	}

	if windowsDesc == nil {
		t.Fatal("expected a Windows node descriptor")
	}
	if windowsDesc.Metadata["kind"] != "windows" {
		t.Fatalf("unexpected Windows node metadata: %+v", windowsDesc.Metadata)
	}

	if wslDesc == nil {
		t.Fatal("expected a WSL node descriptor")
	}
	if wslDesc.ID != "desktop-node-wsl-ubuntu-22-04" {
		t.Fatalf("unexpected WSL node id: %q", wslDesc.ID)
	}
	if wslDesc.Name != "Desktop Node / WSL Ubuntu-22.04" {
		t.Fatalf("unexpected WSL node name: %q", wslDesc.Name)
	}
	if wslDesc.Metadata["kind"] != "wsl" || wslDesc.Metadata["parent_node_id"] != "desktop-node" || wslDesc.Metadata["wsl_distro"] != "Ubuntu-22.04" {
		t.Fatalf("unexpected WSL metadata: %+v", wslDesc.Metadata)
	}

	var allowCommands []string
	if err := json.Unmarshal([]byte(wslDesc.Metadata["approval_allow_command_prefixes"]), &allowCommands); err != nil {
		t.Fatalf("decode command allowlist metadata: %v", err)
	}
	if len(allowCommands) != 1 || allowCommands[0] != "git status" {
		t.Fatalf("unexpected WSL command allowlist: %v", allowCommands)
	}

	var allowPaths []string
	if err := json.Unmarshal([]byte(wslDesc.Metadata["approval_allow_write_path_prefixes"]), &allowPaths); err != nil {
		t.Fatalf("decode path allowlist metadata: %v", err)
	}
	if len(allowPaths) != 1 || allowPaths[0] != "/workspace/project" {
		t.Fatalf("unexpected WSL path allowlist: %v", allowPaths)
	}
}

func TestBuildNodeInstancesFromDistrosErrorsWhenNothingIsEnabled(t *testing.T) {
	_, err := buildNodeInstancesFromDistros(processConfig{
		NodeID:            "desktop-node",
		NodeName:          "Desktop Node",
		EnableWindowsNode: false,
		WSLDistros: []wslDistroConfig{
			{Name: "Ubuntu-22.04", Enabled: true},
		},
	}, "test-host", nil)
	if err == nil {
		t.Fatal("expected an error when no enabled node can be registered")
	}
}

func TestSlugifyDistroName(t *testing.T) {
	if got := slugifyDistroName(" Ubuntu 22.04 "); got != "ubuntu-22-04" {
		t.Fatalf("unexpected slug: %q", got)
	}
}
