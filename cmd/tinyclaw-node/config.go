package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/LittleSongxx/TinyClaw/node"
)

type processConfig struct {
	GatewayWS         string            `json:"gateway_ws"`
	NodeToken         string            `json:"node_token"`
	NodeID            string            `json:"node_id"`
	NodeName          string            `json:"node_name"`
	LogDir            string            `json:"log_dir"`
	StartAtLogin      bool              `json:"start_at_login"`
	EnableWindowsNode bool              `json:"enable_windows_node"`
	WSLDistros        []wslDistroConfig `json:"wsl_distros"`
}

type wslDistroConfig struct {
	Name                   string   `json:"name"`
	Enabled                bool     `json:"enabled"`
	AllowCommandPrefixes   []string `json:"allow_command_prefixes"`
	AllowWritePathPrefixes []string `json:"allow_write_path_prefixes"`
	DefaultCWD             string   `json:"default_cwd"`
}

type fileProcessConfig struct {
	GatewayWS         *string           `json:"gateway_ws"`
	NodeToken         *string           `json:"node_token"`
	NodeID            *string           `json:"node_id"`
	NodeName          *string           `json:"node_name"`
	LogDir            *string           `json:"log_dir"`
	StartAtLogin      *bool             `json:"start_at_login"`
	EnableWindowsNode *bool             `json:"enable_windows_node"`
	WSLDistros        []wslDistroConfig `json:"wsl_distros"`
}

type cliOptions struct {
	ConfigPath string
	Configure  bool
	GatewayWS  string
	NodeID     string
	NodeName   string
	NodeToken  string
}

func parseCLIOptions() (cliOptions, map[string]bool, error) {
	configPath := flag.String("config", defaultConfigPath(), "tinyclaw-node config file path")
	configure := flag.Bool("configure", false, "open the Windows node configuration UI")
	gatewayWS := flag.String("gateway_ws", "", "gateway node websocket endpoint")
	nodeID := flag.String("node_id", "", "node id")
	nodeName := flag.String("node_name", "", "node name")
	nodeToken := flag.String("node_token", "", "node pairing token")

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return cliOptions{}, nil, err
	}

	explicit := make(map[string]bool)
	flag.Visit(func(current *flag.Flag) {
		explicit[current.Name] = true
	})

	return cliOptions{
		ConfigPath: *configPath,
		Configure:  *configure,
		GatewayWS:  *gatewayWS,
		NodeID:     *nodeID,
		NodeName:   *nodeName,
		NodeToken:  *nodeToken,
	}, explicit, nil
}

func resolveProcessConfig(ctx context.Context, opts cliOptions, explicit map[string]bool) (processConfig, error) {
	cfg := defaultProcessConfig(ctx)

	fileCfg, err := loadProcessConfig(opts.ConfigPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return processConfig{}, err
	}
	if err == nil {
		applyFileConfig(&cfg, fileCfg)
	}

	applyCLIOverrides(&cfg, opts, explicit)
	cfg.WSLDistros = mergeDetectedWSLDistros(ctx, cfg.WSLDistros)
	normalizeProcessConfig(&cfg)

	if cfg.GatewayWS == "" {
		return processConfig{}, errors.New("gateway_ws is required")
	}
	if cfg.NodeToken == "" {
		return processConfig{}, errors.New("node_token is required")
	}
	return cfg, nil
}

func defaultProcessConfig(ctx context.Context) processConfig {
	cfg := processConfig{
		GatewayWS:         envOrFallback("NODE_GATEWAY_WS", "ws://127.0.0.1:36060/gateway/nodes/ws"),
		NodeToken:         strings.TrimSpace(os.Getenv("NODE_PAIRING_TOKEN")),
		LogDir:            defaultLogDir(),
		EnableWindowsNode: true,
	}
	cfg.WSLDistros = mergeDetectedWSLDistros(ctx, nil)
	normalizeProcessConfig(&cfg)
	return cfg
}

func loadProcessConfig(path string) (*fileProcessConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var cfg fileProcessConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyFileConfig(target *processConfig, source *fileProcessConfig) {
	if target == nil || source == nil {
		return
	}
	if source.GatewayWS != nil {
		target.GatewayWS = strings.TrimSpace(*source.GatewayWS)
	}
	if source.NodeToken != nil {
		target.NodeToken = strings.TrimSpace(*source.NodeToken)
	}
	if source.NodeID != nil {
		target.NodeID = strings.TrimSpace(*source.NodeID)
	}
	if source.NodeName != nil {
		target.NodeName = strings.TrimSpace(*source.NodeName)
	}
	if source.LogDir != nil {
		target.LogDir = strings.TrimSpace(*source.LogDir)
	}
	if source.StartAtLogin != nil {
		target.StartAtLogin = *source.StartAtLogin
	}
	if source.EnableWindowsNode != nil {
		target.EnableWindowsNode = *source.EnableWindowsNode
	}
	if source.WSLDistros != nil {
		target.WSLDistros = cloneWSLDistroConfigs(source.WSLDistros)
	}
}

func applyCLIOverrides(target *processConfig, opts cliOptions, explicit map[string]bool) {
	if target == nil {
		return
	}
	if explicit["gateway_ws"] {
		target.GatewayWS = strings.TrimSpace(opts.GatewayWS)
	}
	if explicit["node_token"] {
		target.NodeToken = strings.TrimSpace(opts.NodeToken)
	}
	if explicit["node_id"] {
		target.NodeID = strings.TrimSpace(opts.NodeID)
	}
	if explicit["node_name"] {
		target.NodeName = strings.TrimSpace(opts.NodeName)
	}
}

func normalizeProcessConfig(cfg *processConfig) {
	if cfg == nil {
		return
	}
	cfg.GatewayWS = strings.TrimSpace(cfg.GatewayWS)
	cfg.NodeToken = strings.TrimSpace(cfg.NodeToken)
	cfg.NodeID = strings.TrimSpace(cfg.NodeID)
	cfg.NodeName = strings.TrimSpace(cfg.NodeName)
	cfg.LogDir = strings.TrimSpace(cfg.LogDir)
	if cfg.LogDir == "" {
		cfg.LogDir = defaultLogDir()
	}
	if cfg.NodeID == "" {
		if hostname, err := os.Hostname(); err == nil {
			cfg.NodeID = hostname
		}
	}
	if cfg.NodeName == "" {
		cfg.NodeName = cfg.NodeID
	}
	for index := range cfg.WSLDistros {
		cfg.WSLDistros[index].Name = strings.TrimSpace(cfg.WSLDistros[index].Name)
		cfg.WSLDistros[index].DefaultCWD = strings.TrimSpace(cfg.WSLDistros[index].DefaultCWD)
		cfg.WSLDistros[index].AllowCommandPrefixes = normalizePrefixList(cfg.WSLDistros[index].AllowCommandPrefixes, normalizeCommandPrefix)
		cfg.WSLDistros[index].AllowWritePathPrefixes = normalizePrefixList(cfg.WSLDistros[index].AllowWritePathPrefixes, normalizePathPrefix)
	}
}

func mergeDetectedWSLDistros(ctx context.Context, existing []wslDistroConfig) []wslDistroConfig {
	items := cloneWSLDistroConfigs(existing)
	indexByName := make(map[string]int, len(items))
	for index, item := range items {
		indexByName[strings.ToLower(strings.TrimSpace(item.Name))] = index
	}

	distros, err := node.ListWSLDistros(ctx)
	if err != nil {
		return items
	}
	for _, distro := range distros {
		key := strings.ToLower(strings.TrimSpace(distro))
		if key == "" {
			continue
		}
		if _, ok := indexByName[key]; ok {
			continue
		}
		items = append(items, wslDistroConfig{Name: distro})
		indexByName[key] = len(items) - 1
	}
	return items
}

func cloneWSLDistroConfigs(items []wslDistroConfig) []wslDistroConfig {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]wslDistroConfig, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, wslDistroConfig{
			Name:                   item.Name,
			Enabled:                item.Enabled,
			AllowCommandPrefixes:   append([]string(nil), item.AllowCommandPrefixes...),
			AllowWritePathPrefixes: append([]string(nil), item.AllowWritePathPrefixes...),
			DefaultCWD:             item.DefaultCWD,
		})
	}
	return cloned
}

func normalizePrefixList(values []string, normalize func(string) string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = normalize(strings.TrimSpace(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func normalizeCommandPrefix(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func normalizePathPrefix(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return path.Clean(value)
}

func envOrFallback(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func defaultConfigPath() string {
	if runtime.GOOS == "windows" {
		programData := strings.TrimSpace(os.Getenv("ProgramData"))
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "TinyClawNode", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "tinyclaw-node.json"
	}
	return filepath.Join(home, ".config", "tinyclaw-node", "config.json")
}

func defaultLogDir() string {
	if runtime.GOOS == "windows" {
		programData := strings.TrimSpace(os.Getenv("ProgramData"))
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "TinyClawNode", "logs")
	}
	return filepath.Join(".", "log")
}
