package agentruntime

import (
	"context"

	"github.com/LittleSongxx/TinyClaw/tooling"
)

type Mode string

const (
	ModeChat     Mode = "chat"
	ModeTask     Mode = "task"
	ModeMCP      Mode = "mcp"
	ModeSkill    Mode = "skill"
	ModeWorkflow Mode = "workflow"
)

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusSucceeded RunStatus = "succeeded"
	RunStatusFailed    RunStatus = "failed"
)

type StepKind string

const (
	StepKindPlanner     StepKind = "planner"
	StepKindExecutor    StepKind = "executor"
	StepKindJudge       StepKind = "judge"
	StepKindSynthesizer StepKind = "synthesizer"
)

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusSucceeded StepStatus = "succeeded"
	StepStatusFailed    StepStatus = "failed"
)

type RunMeta struct {
	UserID   string
	ChatID   string
	MsgID    string
	Input    string
	Mode     Mode
	ReplayOf int64
	SkillID  string
}

type TaskPlan struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type StepEvidence struct {
	StepIndex    int                   `json:"step_index"`
	Name         string                `json:"name"`
	ToolName     string                `json:"tool_name"`
	Prompt       string                `json:"prompt"`
	Output       string                `json:"output"`
	Observations []tooling.Observation `json:"observations,omitempty"`
}

type Planner interface {
	PlanTask(ctx context.Context, input string, tools []tooling.ToolSpec) ([]TaskPlan, string, int, error)
	SelectTool(ctx context.Context, input string, tools []tooling.ToolSpec) (string, string, int, error)
	JudgeTask(ctx context.Context, input string, evidence []StepEvidence, lastPlan string, tools []tooling.ToolSpec) ([]TaskPlan, string, int, error)
}

type Executor interface {
	RespondDirect(ctx context.Context, input string) (string, int, error)
	ExecuteStep(ctx context.Context, input string, entry *tooling.Entry) (*tooling.ToolResult, int, error)
	ExecuteDirect(ctx context.Context, input string, entry *tooling.Entry) (*tooling.ToolResult, int, error)
	Synthesize(ctx context.Context, input string, evidence []StepEvidence) (string, int, error)
}
