package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/LittleSongxx/TinyClaw/gateway"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/gorilla/websocket"
)

const nodeBinaryVersion = "v0.2.0"

type nodeInstance struct {
	descriptor *node.NodeDescriptor
	driver     node.Driver
}

func runConfigureMode(configPath string) error {
	scriptPath, err := resolveConfigureScriptPath()
	if err != nil {
		return err
	}
	executable, err := nodePowerShellExecutable()
	if err != nil {
		return err
	}

	cmd := exec.Command(executable,
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
		"-ConfigPath", configPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func resolveConfigureScriptPath() (string, error) {
	candidates := make([]string, 0, 3)
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "configure-node.ps1"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "deploy", "windows-node", "configure-node.ps1"),
			filepath.Join(cwd, "configure-node.ps1"),
		)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("configure-node.ps1 not found next to the executable or in deploy/windows-node")
}

func buildNodeInstances(ctx context.Context, cfg processConfig) ([]nodeInstance, error) {
	hostname := hostName()
	availableDistros, err := node.ListWSLDistros(ctx)
	if err != nil {
		logger.Warn("list wsl distros failed", "err", err)
		availableDistros = nil
	}
	return buildNodeInstancesFromDistros(cfg, hostname, availableDistros)
}

func buildNodeInstancesFromDistros(cfg processConfig, hostname string, availableDistros []string) ([]nodeInstance, error) {
	items := make([]nodeInstance, 0, 1+len(cfg.WSLDistros))
	if cfg.EnableWindowsNode {
		driver := node.NewLocalDriver()
		items = append(items, nodeInstance{
			descriptor: &node.NodeDescriptor{
				ID:           cfg.NodeID,
				Name:         cfg.NodeName,
				Platform:     runtimePlatform(),
				Hostname:     hostname,
				Version:      nodeBinaryVersion,
				Metadata:     map[string]string{"kind": "windows"},
				Capabilities: driver.Capabilities(),
			},
			driver: driver,
		})
	}

	available := make(map[string]string, len(availableDistros))
	for _, distro := range availableDistros {
		available[strings.ToLower(strings.TrimSpace(distro))] = distro
	}

	for _, distro := range cfg.WSLDistros {
		if !distro.Enabled {
			continue
		}
		resolvedName := available[strings.ToLower(strings.TrimSpace(distro.Name))]
		if resolvedName == "" {
			logger.Warn("configured wsl distro is unavailable, skipping", "distro", distro.Name)
			continue
		}

		driver := node.NewWSLDriver(node.WSLDriverConfig{
			DistroName: resolvedName,
			DefaultCWD: distro.DefaultCWD,
		})
		items = append(items, nodeInstance{
			descriptor: &node.NodeDescriptor{
				ID:       cfg.NodeID + "-wsl-" + slugifyDistroName(resolvedName),
				Name:     cfg.NodeName + " / WSL " + resolvedName,
				Platform: "wsl",
				Hostname: hostname,
				Version:  nodeBinaryVersion,
				Metadata: map[string]string{
					"kind":                               "wsl",
					"parent_node_id":                     cfg.NodeID,
					"wsl_distro":                         resolvedName,
					"approval_allow_command_prefixes":    encodeStringSlice(distro.AllowCommandPrefixes),
					"approval_allow_write_path_prefixes": encodeStringSlice(distro.AllowWritePathPrefixes),
				},
				Capabilities: driver.Capabilities(),
			},
			driver: driver,
		})
	}

	if len(items) == 0 {
		return nil, errors.New("no Windows or WSL nodes are enabled")
	}
	return items, nil
}

func runNodeLoop(ctx context.Context, gatewayWS, token string, instance nodeInstance) {
	for {
		if err := runNode(ctx, gatewayWS, token, instance.descriptor, instance.driver); err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Warn("tinyclaw-node disconnected, retrying",
				"node_id", instance.descriptor.ID,
				"platform", instance.descriptor.Platform,
				"err", err,
			)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}
}

func runtimePlatform() string {
	return strings.TrimSpace(strings.ToLower(runtime.GOOS))
}

var distroSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugifyDistroName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = distroSlugPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "default"
	}
	return value
}

func encodeStringSlice(values []string) string {
	if len(values) == 0 {
		return ""
	}
	content, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return string(content)
}

func nodePowerShellExecutable() (string, error) {
	candidates := []string{"powershell.exe", "powershell", "pwsh.exe", "pwsh"}
	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate)
		if err == nil {
			return path, nil
		}
	}
	return "", errors.New("powershell executable is not available")
}

func runNode(ctx context.Context, gatewayWS, token string, descriptor *node.NodeDescriptor, driver node.Driver) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, gatewayWS, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	var writeMu sync.Mutex

	connectFrame := gateway.NewConnectFrame("node", token, descriptor)
	if err := writeJSON(&writeMu, conn, connectFrame); err != nil {
		return err
	}

	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatTicker.C:
				frame, err := gateway.NewEventFrame("node.heartbeat", map[string]interface{}{
					"node_id": descriptor.ID,
				})
				if err == nil {
					_ = writeJSON(&writeMu, conn, frame)
				}
			}
		}
	}()

	for {
		var request gateway.RequestFrame
		if err := conn.ReadJSON(&request); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if request.Action != "node.command" {
			response, respErr := gateway.NewResponseFrame(request.ID, false, nil, "unsupported node action")
			if respErr == nil {
				_ = writeJSON(&writeMu, conn, response)
			}
			continue
		}

		var command node.NodeCommandRequest
		if err := json.Unmarshal(request.Payload, &command); err != nil {
			response, respErr := gateway.NewResponseFrame(request.ID, false, nil, err.Error())
			if respErr == nil {
				_ = writeJSON(&writeMu, conn, response)
			}
			continue
		}
		if command.ID == "" {
			command.ID = request.ID
		}
		command.NodeID = descriptor.ID

		result, execErr := driver.Execute(ctx, command)
		response, respErr := gateway.NewResponseFrame(request.ID, execErr == nil, result, errorText(execErr))
		if respErr != nil {
			return respErr
		}
		if err := writeJSON(&writeMu, conn, response); err != nil {
			return err
		}
	}
}
