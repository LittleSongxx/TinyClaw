package llm

import (
	"context"

	"github.com/LittleSongxx/TinyClaw/agentruntime"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/param"
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

	Cs  *param.ContextState
	Ctx context.Context
}

// ExecuteTask execute task command
func (d *LLMTaskReq) ExecuteTask() error {
	_, err := d.ExecuteTaskRun()
	return err
}

func (d *LLMTaskReq) ExecuteTaskRun() (*db.AgentRun, error) {
	runner := d.newRunner()
	return runner.RunTask(d.runtimeContext(), d.runMeta(agentruntime.ModeTask))
}

func (d *LLMTaskReq) newRunner() *agentruntime.Runner {
	adapter := newRuntimeAdapter(d)
	return &agentruntime.Runner{
		Planner:  adapter,
		Executor: adapter,
		Registry: tooling.NewRegistryFromTaskTools(),
		MaxSteps: MostLoop,
	}
}

func (d *LLMTaskReq) runMeta(mode agentruntime.Mode) agentruntime.RunMeta {
	return agentruntime.RunMeta{
		UserID:   d.UserId,
		ChatID:   d.ChatId,
		MsgID:    d.MsgId,
		Input:    d.Content,
		Mode:     mode,
		ReplayOf: d.ReplayOf,
	}
}

func (d *LLMTaskReq) runtimeContext() context.Context {
	if d.Ctx != nil {
		return d.Ctx
	}
	return context.Background()
}
