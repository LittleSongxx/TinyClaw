package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
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
	WorkspaceID       string            `json:"workspace_id"`
	DeviceID          string            `json:"device_id"`
	DeviceToken       string            `json:"device_token"`
	PrivateKey        string            `json:"private_key"`
	PublicKey         string            `json:"public_key"`
	PairingCode       string            `json:"pairing_code,omitempty"`
	NodeName          string            `json:"node_name"`
	DeprecatedNodeID  string            `json:"-"`
	DeprecatedNodeToken string          `json:"-"`
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
	WorkspaceID       *string           `json:"workspace_id"`
	DeviceID          *string           `json:"device_id"`
	DeviceToken       *string           `json:"device_token"`
	PrivateKey        *string           `json:"private_key"`
	PublicKey         *string           `json:"public_key"`
	PairingCode       *string           `json:"pairing_code"`
	DeprecatedNodeToken *string         `json:"node_token"`
	DeprecatedNodeID  *string           `json:"node_id"`
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
	WorkspaceID string
	DeviceID   string
	NodeName   string
	DeviceToken string
	PairingCode string
}

func parseCLIOptions() (cliOptions, map[string]bool, error) {
	configPath := flag.String("config", defaultConfigPath(), "tinyclaw-node config file path")
	configure := flag.Bool("configure", false, "open the Windows node configuration UI")
	gatewayWS := flag.String("gateway_ws", "", "gateway node websocket endpoint")
	workspaceID := flag.String("workspace_id", "", "workspace id")
	deviceID := flag.String("device_id", "", "device id")
	nodeName := flag.String("node_name", "", "node name")
	deviceToken := flag.String("device_token", "", "device token")
	pairingCode := flag.String("pairing_code", "", "10 minute device bootstrap pairing code")

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
		WorkspaceID: *workspaceID,
		DeviceID:   *deviceID,
		NodeName:   *nodeName,
		DeviceToken: *deviceToken,
		PairingCode: *pairingCode,
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

	if cfg.DeprecatedNodeToken != "" || strings.TrimSpace(os.Getenv("NODE_PAIRING_TOKEN")) != "" {
		return processConfig{}, errors.New("static node_token/NODE_PAIRING_TOKEN is no longer supported; run Device Pairing and configure device_token/private_key/public_key")
	}
	if cfg.GatewayWS == "" {
		return processConfig{}, errors.New("gateway_ws is required")
	}
	if cfg.WorkspaceID == "" {
		return processConfig{}, errors.New("workspace_id is required")
	}
	if cfg.DeviceID == "" {
		return processConfig{}, errors.New("device_id is required")
	}
	if cfg.DeviceToken == "" && cfg.PairingCode == "" {
		return processConfig{}, errors.New("device_token is required; use pairing_code only for initial pairing")
	}
	if cfg.PublicKey == "" || cfg.PrivateKey == "" {
		return processConfig{}, errors.New("private_key and public_key are required")
	}
	return cfg, nil
}

func defaultProcessConfig(ctx context.Context) processConfig {
	cfg := processConfig{
		GatewayWS:         envOrFallback("NODE_GATEWAY_WS", "ws://127.0.0.1:36060/gateway/nodes/ws"),
		WorkspaceID:       envOrFallback("TINYCLAW_WORKSPACE_ID", "default"),
		DeviceID:          strings.TrimSpace(os.Getenv("TINYCLAW_DEVICE_ID")),
		DeviceToken:       strings.TrimSpace(os.Getenv("TINYCLAW_DEVICE_TOKEN")),
		PrivateKey:        strings.TrimSpace(os.Getenv("TINYCLAW_DEVICE_PRIVATE_KEY")),
		PublicKey:         strings.TrimSpace(os.Getenv("TINYCLAW_DEVICE_PUBLIC_KEY")),
		PairingCode:       strings.TrimSpace(os.Getenv("TINYCLAW_PAIRING_CODE")),
		DeprecatedNodeToken: strings.TrimSpace(os.Getenv("NODE_PAIRING_TOKEN")),
		LogDir:            defaultLogDir(),
		EnableWindowsNode: true,
	}
	if cfg.PrivateKey == "" || cfg.PublicKey == "" {
		cfg.PrivateKey, cfg.PublicKey = generateDeviceKeyPair()
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
	if source.WorkspaceID != nil {
		target.WorkspaceID = strings.TrimSpace(*source.WorkspaceID)
	}
	if source.DeviceID != nil {
		target.DeviceID = strings.TrimSpace(*source.DeviceID)
	}
	if source.DeviceToken != nil {
		target.DeviceToken = strings.TrimSpace(*source.DeviceToken)
	}
	if source.PrivateKey != nil {
		target.PrivateKey = strings.TrimSpace(*source.PrivateKey)
	}
	if source.PublicKey != nil {
		target.PublicKey = strings.TrimSpace(*source.PublicKey)
	}
	if source.PairingCode != nil {
		target.PairingCode = strings.TrimSpace(*source.PairingCode)
	}
	if source.DeprecatedNodeToken != nil {
		target.DeprecatedNodeToken = strings.TrimSpace(*source.DeprecatedNodeToken)
	}
	if source.DeprecatedNodeID != nil && target.DeviceID == "" {
		target.DeprecatedNodeID = strings.TrimSpace(*source.DeprecatedNodeID)
		target.DeviceID = target.DeprecatedNodeID
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
	if explicit["workspace_id"] {
		target.WorkspaceID = strings.TrimSpace(opts.WorkspaceID)
	}
	if explicit["device_id"] {
		target.DeviceID = strings.TrimSpace(opts.DeviceID)
	}
	if explicit["device_token"] {
		target.DeviceToken = strings.TrimSpace(opts.DeviceToken)
	}
	if explicit["pairing_code"] {
		target.PairingCode = strings.TrimSpace(opts.PairingCode)
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
	cfg.WorkspaceID = strings.TrimSpace(cfg.WorkspaceID)
	if cfg.WorkspaceID == "" {
		cfg.WorkspaceID = "default"
	}
	cfg.DeviceID = strings.TrimSpace(cfg.DeviceID)
	cfg.DeviceToken = strings.TrimSpace(cfg.DeviceToken)
	cfg.PrivateKey = strings.TrimSpace(cfg.PrivateKey)
	cfg.PublicKey = strings.TrimSpace(cfg.PublicKey)
	cfg.PairingCode = strings.TrimSpace(cfg.PairingCode)
	cfg.DeprecatedNodeToken = strings.TrimSpace(cfg.DeprecatedNodeToken)
	cfg.NodeName = strings.TrimSpace(cfg.NodeName)
	cfg.LogDir = strings.TrimSpace(cfg.LogDir)
	if cfg.LogDir == "" {
		cfg.LogDir = defaultLogDir()
	}
	if cfg.DeviceID == "" {
		if hostname, err := os.Hostname(); err == nil {
			cfg.DeviceID = hostname
		}
	}
	if cfg.NodeName == "" {
		cfg.NodeName = cfg.DeviceID
	}
	for index := range cfg.WSLDistros {
		cfg.WSLDistros[index].Name = strings.TrimSpace(cfg.WSLDistros[index].Name)
		cfg.WSLDistros[index].DefaultCWD = strings.TrimSpace(cfg.WSLDistros[index].DefaultCWD)
		cfg.WSLDistros[index].AllowCommandPrefixes = normalizePrefixList(cfg.WSLDistros[index].AllowCommandPrefixes, normalizeCommandPrefix)
		cfg.WSLDistros[index].AllowWritePathPrefixes = normalizePrefixList(cfg.WSLDistros[index].AllowWritePathPrefixes, normalizePathPrefix)
	}
}

func generateDeviceKeyPair() (string, string) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", ""
	}
	return base64.RawStdEncoding.EncodeToString(privateKey), base64.RawStdEncoding.EncodeToString(publicKey)
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
