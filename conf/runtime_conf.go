package conf

import "os"

type GatewayConf struct {
	Enabled      bool   `json:"enabled"`
	ClientWSPath string `json:"client_ws_path"`
	NodeWSPath   string `json:"node_ws_path"`
	SharedSecret string `json:"-"`
}

type SessionConf struct {
	Enabled       bool   `json:"enabled"`
	TranscriptDir string `json:"transcript_dir"`
	ContextWindow int    `json:"context_window"`
}

type NodeRuntimeConf struct {
	Enabled                  bool   `json:"enabled"`
	LegacyNodeTokenPresent   bool   `json:"-"`
	DefaultCommandTimeoutSec int    `json:"default_command_timeout_sec"`
}

type RuntimeConfig struct {
	Gateway  GatewayConf     `json:"gateway"`
	Sessions SessionConf     `json:"sessions"`
	Nodes    NodeRuntimeConf `json:"nodes"`
}

var RuntimeConfInfo = &RuntimeConfig{}

func InitRuntimeConf() {
	RuntimeConfInfo.Gateway.Enabled = true
	RuntimeConfInfo.Gateway.ClientWSPath = envOrDefault("GATEWAY_WS_PATH", "/gateway/ws")
	RuntimeConfInfo.Gateway.NodeWSPath = envOrDefault("GATEWAY_NODE_WS_PATH", "/gateway/nodes/ws")
	RuntimeConfInfo.Gateway.SharedSecret = os.Getenv("GATEWAY_SHARED_SECRET")

	RuntimeConfInfo.Sessions.Enabled = true
	RuntimeConfInfo.Sessions.TranscriptDir = envOrDefault("SESSION_TRANSCRIPT_DIR", GetAbsPath("data/sessions"))
	RuntimeConfInfo.Sessions.ContextWindow = 20

	RuntimeConfInfo.Nodes.Enabled = true
	RuntimeConfInfo.Nodes.LegacyNodeTokenPresent = os.Getenv("NODE_PAIRING_TOKEN") != ""
	RuntimeConfInfo.Nodes.DefaultCommandTimeoutSec = 30
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
