package llm

import (
	"context"

	"github.com/LittleSongxx/TinyClaw/agentruntime"
	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/skill"
	"github.com/LittleSongxx/TinyClaw/tooling"
)

type LLMTaskReq struct {
	MessageChan chan *param.MsgInfo
	HTTPMsgChan chan string
	Content     string
	Model       string
	Token       int
	PerMsgLen   int

	UserId   string
	ChatId   string
	MsgId    string
	ReplayOf int64
	SkillID  string

	Cs  *param.ContextState
	Ctx context.Context
}

// ExecuteTask execute task command
func (d *LLMTaskReq) ExecuteTask() error {
	_, err := d.ExecuteTaskRun()
	return err
}

func (d *LLMTaskReq) ExecuteTaskRun() (*db.AgentRun, error) {
	runner := d.newRunner(agentruntime.ModeTask)
	return runner.RunTask(d.runtimeContext(), d.runMeta(agentruntime.ModeTask))
}

func (d *LLMTaskReq) ExecuteSkill() error {
	_, err := d.ExecuteSkillRun()
	return err
}

func (d *LLMTaskReq) ExecuteSkillRun() (*db.AgentRun, error) {
	runner := d.newRunner(agentruntime.ModeSkill)
	return runner.RunSkill(d.runtimeContext(), d.runMeta(agentruntime.ModeSkill))
}

func (d *LLMTaskReq) newRunner(mode agentruntime.Mode) *agentruntime.Runner {
	adapter := newRuntimeAdapter(d)
	registry := d.newRuntimeRegistry(mode)
	return &agentruntime.Runner{
		Planner:  adapter,
		Executor: adapter,
		Registry: registry,
		MaxSteps: MostLoop,
	}
}

func (d *LLMTaskReq) newRuntimeRegistry(mode agentruntime.Mode) *tooling.Registry {
	catalog, err := skill.LoadCatalog(skill.LoadOptions{
		SkillRoots: skill.DefaultRoots(),
		MCPConfPath: func() string {
			if conf.ToolsConfInfo == nil || conf.ToolsConfInfo.McpConfPath == nil {
				return ""
			}
			return *conf.ToolsConfInfo.McpConfPath
		}(),
	})
	if err != nil {
		logger.WarnCtx(d.runtimeContext(), "load skill catalog fail", "err", err)
	}
	if catalog != nil {
		maxCandidates := 8
		if mode == agentruntime.ModeSkill {
			maxCandidates = 1
		}
		registry := catalog.BuildRegistry(string(mode), d.Content, d.SkillID, maxCandidates)
		if registry != nil && len(registry.List()) > 0 {
			return registry
		}
	}

	if conf.FeatureConfInfo.LegacyTaskToolsEnabled() {
		return tooling.NewRegistryFromTaskTools()
	}
	return tooling.NewRegistry()
}

func (d *LLMTaskReq) runMeta(mode agentruntime.Mode) agentruntime.RunMeta {
	return agentruntime.RunMeta{
		UserID:   d.UserId,
		ChatID:   d.ChatId,
		MsgID:    d.MsgId,
		Input:    d.Content,
		Mode:     mode,
		ReplayOf: d.ReplayOf,
		SkillID:  d.SkillID,
	}
}

func (d *LLMTaskReq) runtimeContext() context.Context {
	if d.Ctx != nil {
		return d.Ctx
	}
	return context.Background()
}
