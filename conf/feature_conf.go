package conf

import (
	"flag"
	"os"
	"strings"
)

type FeatureConf struct {
	Knowledge       bool `json:"knowledge"`
	Media           bool `json:"media"`
	Cron            bool `json:"cron"`
	LegacyBots      bool `json:"legacy_bots"`
	LegacyMCPProxy  bool `json:"legacy_mcp_proxy"`
	LegacyTaskTools bool `json:"legacy_task_tools"`
	Workflow        bool `json:"workflow"`
}

var FeatureConfInfo = new(FeatureConf)

func InitFeatureConf() {
	flag.BoolVar(&FeatureConfInfo.Knowledge, "enable_knowledge", false, "enable optional knowledge/RAG module")
	flag.BoolVar(&FeatureConfInfo.Media, "enable_media", false, "enable optional image/video/audio commands")
	flag.BoolVar(&FeatureConfInfo.Cron, "enable_cron", false, "enable optional cron commands and scheduler")
	flag.BoolVar(&FeatureConfInfo.LegacyBots, "enable_legacy_bots", false, "enable legacy chat platform adapters")
	flag.BoolVar(&FeatureConfInfo.LegacyMCPProxy, "enable_legacy_mcp_proxy", false, "enable generated legacy MCP proxy skills")
	flag.BoolVar(&FeatureConfInfo.LegacyTaskTools, "enable_legacy_task_tools", false, "enable legacy MCP server-level task tools")
	flag.BoolVar(&FeatureConfInfo.Workflow, "enable_experimental_workflow", false, "enable experimental workflow mode")
}

func EnvFeatureConf() {
	FeatureConfInfo.Knowledge = envBool("ENABLE_KNOWLEDGE", FeatureConfInfo.Knowledge)
	FeatureConfInfo.Media = envBool("ENABLE_MEDIA", FeatureConfInfo.Media)
	FeatureConfInfo.Cron = envBool("ENABLE_CRON", FeatureConfInfo.Cron)
	FeatureConfInfo.LegacyBots = envBool("ENABLE_LEGACY_BOTS", FeatureConfInfo.LegacyBots)
	FeatureConfInfo.LegacyMCPProxy = envBool("ENABLE_LEGACY_MCP_PROXY", FeatureConfInfo.LegacyMCPProxy)
	FeatureConfInfo.LegacyTaskTools = envBool("ENABLE_LEGACY_TASK_TOOLS", FeatureConfInfo.LegacyTaskTools)
	FeatureConfInfo.Workflow = envBool("ENABLE_EXPERIMENTAL_WORKFLOW", FeatureConfInfo.Workflow)
}

func (r *FeatureConf) KnowledgeEnabled() bool {
	return r != nil && r.Knowledge
}

func (r *FeatureConf) MediaEnabled() bool {
	return r != nil && r.Media
}

func (r *FeatureConf) CronEnabled() bool {
	return r != nil && r.Cron
}

func (r *FeatureConf) LegacyBotsEnabled() bool {
	return r != nil && r.LegacyBots
}

func (r *FeatureConf) LegacyMCPProxyEnabled() bool {
	return r != nil && r.LegacyMCPProxy
}

func (r *FeatureConf) LegacyTaskToolsEnabled() bool {
	return r != nil && r.LegacyTaskTools
}

func (r *FeatureConf) WorkflowEnabled() bool {
	return r != nil && r.Workflow
}

func envBool(name string, fallback bool) bool {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "true" || value == "1" || value == "yes" || value == "on"
}
