package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LittleSongxx/TinyClaw/agentruntime"
	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/i18n"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/skill"
	"github.com/LittleSongxx/TinyClaw/tooling"
	"github.com/LittleSongxx/TinyClaw/utils"
)

type runtimeAdapter struct {
	req *LLMTaskReq
}

type taskPlanEnvelope struct {
	Plan []agentruntime.TaskPlan `json:"plan"`
}

type toolSelectionEnvelope struct {
	Agent string `json:"agent"`
}

func newRuntimeAdapter(req *LLMTaskReq) *runtimeAdapter {
	if req.Cs == nil {
		req.Cs = &param.ContextState{UseRecord: true}
	}
	return &runtimeAdapter{req: req}
}

func (a *runtimeAdapter) StepMetadata() (string, string) {
	if a == nil || a.req == nil {
		return conf.BaseConfInfo.Type, ""
	}

	var llmConf *param.LLMConfig
	if userInfo := db.GetCtxUserInfo(a.req.runtimeContext()); userInfo != nil {
		llmConf = userInfo.LLMConfigRaw
	}

	provider := utils.GetTxtType(llmConf)
	probe := NewLLM(a.baseOptions("", false)...)
	probe.LLMClient.GetModel(probe)
	return provider, probe.Model
}

func (a *runtimeAdapter) PlanTask(ctx context.Context, input string, tools []tooling.ToolSpec) ([]agentruntime.TaskPlan, string, int, error) {
	prompt := i18n.GetMessage("assign_task_prompt", a.promptArgs(input, tools))
	output, token, err := a.executeSilent(ctx, prompt)
	if err != nil {
		return nil, output, token, err
	}

	plans, parseErr := parseTaskPlans(output)
	if parseErr != nil {
		logger.WarnCtx(a.req.runtimeContext(), "parse task plan fail", "err", parseErr, "output", output)
		return nil, output, token, nil
	}

	return plans, output, token, nil
}

func (a *runtimeAdapter) SelectTool(ctx context.Context, input string, tools []tooling.ToolSpec) (string, string, int, error) {
	prompt := i18n.GetMessage("mcp_prompt", a.promptArgs(input, tools))
	output, token, err := a.executeSilent(ctx, prompt)
	if err != nil {
		return "", output, token, err
	}

	toolName, parseErr := parseSelectedTool(output)
	if parseErr != nil {
		logger.WarnCtx(a.req.runtimeContext(), "parse tool selection fail", "err", parseErr, "output", output)
		return "", output, token, nil
	}

	return toolName, output, token, nil
}

func (a *runtimeAdapter) JudgeTask(ctx context.Context, input string, evidence []agentruntime.StepEvidence, lastPlan string, tools []tooling.ToolSpec) ([]agentruntime.TaskPlan, string, int, error) {
	taskParam := a.promptArgs(input, tools)
	completeTasks := make(map[string]bool, len(evidence))
	for _, step := range evidence {
		completeTasks[step.Prompt] = true
	}

	taskParam["complete_tasks"] = completeTasks
	taskParam["last_plan"] = lastPlan
	prompt := i18n.GetMessage("loop_task_prompt", taskParam)
	if len(evidence) > 0 {
		rawEvidence, err := json.MarshalIndent(evidence, "", "  ")
		if err == nil {
			prompt += "\n\nTask execution evidence(JSON):\n" + string(rawEvidence)
		}
	}

	output, token, err := a.executeSilent(ctx, prompt)
	if err != nil {
		return nil, output, token, err
	}

	plans, parseErr := parseTaskPlans(output)
	if parseErr != nil {
		logger.WarnCtx(a.req.runtimeContext(), "parse next task plan fail", "err", parseErr, "output", output)
		return nil, output, token, nil
	}

	return plans, output, token, nil
}

func (a *runtimeAdapter) RespondDirect(ctx context.Context, input string) (string, int, error) {
	return a.executeVisible(ctx, input, input)
}

func (a *runtimeAdapter) ExecuteStep(ctx context.Context, input string, entry *tooling.Entry) (*tooling.ToolResult, int, error) {
	return a.executeWithTool(ctx, input, input, entry, false)
}

func (a *runtimeAdapter) ExecuteDirect(ctx context.Context, input string, entry *tooling.Entry) (*tooling.ToolResult, int, error) {
	return a.executeWithTool(ctx, input, input, entry, true)
}

func (a *runtimeAdapter) Synthesize(ctx context.Context, input string, evidence []agentruntime.StepEvidence) (string, int, error) {
	summaryParam := map[string]interface{}{"user_task": input}
	prompt := i18n.GetMessage("summary_task_prompt", summaryParam)
	if len(evidence) > 0 {
		rawEvidence, err := json.MarshalIndent(evidence, "", "  ")
		if err == nil {
			prompt += "\n\nCollected task evidence(JSON):\n" + string(rawEvidence)
		}
	}

	return a.executeVisible(ctx, prompt, input)
}

func (a *runtimeAdapter) executeWithTool(ctx context.Context, prompt, recordInput string, entry *tooling.Entry, visible bool) (*tooling.ToolResult, int, error) {
	if entry == nil || entry.AgentInfo == nil {
		return nil, 0, fmt.Errorf("tool entry is invalid")
	}

	runCtx, cancel := a.toolContext(ctx, entry)
	defer cancel()

	execPrompt := prompt
	allowedTools := make([]string, 0)
	memoryObservations := make([]tooling.Observation, 0)
	if entry.Skill != nil {
		memoryContext, observations := skill.LoadMemoryContext(runCtx, a.req.UserId, prompt, entry)
		memoryObservations = append(memoryObservations, observations...)
		execPrompt = skill.BuildPromptWithMemory(entry, prompt, memoryContext)
		allowedTools = append(allowedTools, entry.Skill.AllowedTools...)
	}

	startedAt := time.Now().Unix()
	observations := append([]tooling.Observation(nil), memoryObservations...)
	var lock sync.Mutex

	observer := func(obs tooling.Observation) {
		lock.Lock()
		defer lock.Unlock()
		observations = append(observations, obs)
	}

	var (
		output string
		token  int
		err    error
	)

	if visible {
		output, token, err = a.executeVisible(runCtx, execPrompt, recordInput, WithTaskTools(entry.AgentInfo), WithToolObserver(observer), WithAllowedToolNames(allowedTools))
	} else {
		output, token, err = a.executeSilent(runCtx, execPrompt, WithTaskTools(entry.AgentInfo), WithToolObserver(observer), WithAllowedToolNames(allowedTools))
	}

	if err == nil && entry.Skill != nil {
		lock.Lock()
		observations = append(observations, skill.PersistMemoryContext(runCtx, a.req.UserId, prompt, output, entry)...)
		lock.Unlock()
	}

	lock.Lock()
	defer lock.Unlock()

	result := &tooling.ToolResult{
		Name:         entry.Spec.Name,
		Output:       output,
		Observations: append([]tooling.Observation(nil), observations...),
		StartedAt:    startedAt,
		CompletedAt:  time.Now().Unix(),
	}
	if err != nil {
		result.Error = err.Error()
	}

	return result, token, err
}

func (a *runtimeAdapter) executeSilent(ctx context.Context, prompt string, extra ...Option) (string, int, error) {
	start := a.tokenSnapshot()
	opts := a.baseOptions(prompt, false)
	opts = append(opts, extra...)

	llm := NewLLM(opts...)
	llm.GetMessages(a.req.UserId, prompt)
	llm.LLMClient.GetModel(llm)

	output, err := llm.LLMClient.SyncSend(ctx, llm)
	return output, a.tokenDelta(start), err
}

func (a *runtimeAdapter) executeVisible(ctx context.Context, prompt, recordInput string, extra ...Option) (string, int, error) {
	start := a.tokenSnapshot()
	opts := a.baseOptions(prompt, true)
	opts = append(opts, extra...)

	llm := NewLLM(opts...)
	llm.GetMessages(a.req.UserId, prompt)
	llm.LLMClient.GetModel(llm)

	var (
		output string
		err    error
	)

	if a.hasOutputChannel() {
		err = llm.LLMClient.Send(ctx, llm)
		output = llm.WholeContent
	} else {
		output, err = llm.LLMClient.SyncSend(ctx, llm)
		llm.WholeContent = output
	}
	if err != nil {
		return output, a.tokenDelta(start), err
	}

	llm.Content = recordInput
	if err = llm.InsertOrUpdate(); err != nil {
		return output, a.tokenDelta(start), err
	}

	return output, a.tokenDelta(start), nil
}

func (a *runtimeAdapter) toolContext(ctx context.Context, entry *tooling.Entry) (context.Context, context.CancelFunc) {
	if entry == nil || entry.Spec.Policy.Timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, entry.Spec.Policy.Timeout)
}

func (a *runtimeAdapter) promptArgs(input string, tools []tooling.ToolSpec) map[string]interface{} {
	assignParam := make([]map[string]string, 0, len(tools))
	for _, tool := range tools {
		allowedTools := strings.Join(tool.AllowedTools, ", ")
		if allowedTools == "" {
			allowedTools = "-"
		}
		triggers := strings.Join(tool.Triggers, ", ")
		if triggers == "" {
			triggers = "-"
		}

		assignParam = append(assignParam, map[string]string{
			"tool_name":             tool.Name,
			"tool_desc":             tool.Description,
			"tool_version":          fallbackString(tool.Version, "v1"),
			"tool_memory":           fallbackString(tool.Memory, "conversation"),
			"tool_when_to_use":      fallbackString(tool.WhenToUse, tool.Description),
			"tool_when_not_to_use":  fallbackString(tool.WhenNotToUse, "No explicit exclusion guidance."),
			"tool_instructions":     fallbackString(tool.Instructions, "Stay focused on the assigned sub-task."),
			"tool_output_contract":  fallbackString(tool.OutputContract, "Return the requested result clearly."),
			"tool_failure_handling": fallbackString(tool.FailureHandling, "Explain blockers and missing information."),
			"tool_allowed_tools":    allowedTools,
			"tool_triggers":         triggers,
			"tool_legacy":           strconv.FormatBool(tool.Legacy),
		})
	}

	return map[string]interface{}{
		"assign_param": assignParam,
		"user_task":    input,
	}
}

func fallbackString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func (a *runtimeAdapter) baseOptions(content string, visible bool) []Option {
	opts := []Option{
		WithUserId(a.req.UserId),
		WithChatId(a.req.ChatId),
		WithMsgId(a.req.MsgId),
		WithContent(content),
		WithPerMsgLen(a.req.PerMsgLen),
		WithContext(a.req.runtimeContext()),
		WithCS(a.req.Cs),
	}

	if visible && a.req.MessageChan != nil {
		opts = append(opts, WithMessageChan(a.req.MessageChan))
	}
	if visible && a.req.HTTPMsgChan != nil {
		opts = append(opts, WithHTTPMsgChan(a.req.HTTPMsgChan))
	}

	return opts
}

func (a *runtimeAdapter) hasOutputChannel() bool {
	return a.req.MessageChan != nil || a.req.HTTPMsgChan != nil
}

func (a *runtimeAdapter) tokenSnapshot() int {
	if a.req.Cs == nil {
		return 0
	}
	return a.req.Cs.Token
}

func (a *runtimeAdapter) tokenDelta(start int) int {
	if a.req.Cs == nil {
		return 0
	}
	return a.req.Cs.Token - start
}

func parseTaskPlans(output string) ([]agentruntime.TaskPlan, error) {
	body, err := agentruntime.ExtractJSONObject(output)
	if err != nil {
		return nil, nil
	}

	var envelope taskPlanEnvelope
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return nil, err
	}

	plans := make([]agentruntime.TaskPlan, 0, len(envelope.Plan))
	for _, plan := range envelope.Plan {
		name := strings.TrimSpace(plan.Name)
		description := strings.TrimSpace(plan.Description)
		if name == "" || description == "" {
			continue
		}
		plans = append(plans, agentruntime.TaskPlan{
			Name:        name,
			Description: description,
		})
	}

	return plans, nil
}

func parseSelectedTool(output string) (string, error) {
	body, err := agentruntime.ExtractJSONObject(output)
	if err != nil {
		return "", nil
	}

	var envelope toolSelectionEnvelope
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return "", err
	}

	return strings.TrimSpace(envelope.Agent), nil
}
