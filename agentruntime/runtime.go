package agentruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/tooling"
)

type Runner struct {
	Planner  Planner
	Executor Executor
	Registry *tooling.Registry
	MaxSteps int
}

func (r *Runner) RunTask(ctx context.Context, meta RunMeta) (*db.AgentRun, error) {
	run := r.startRun(meta)
	tools := r.Registry.List()

	plans, rawPlan, token, err := r.Planner.PlanTask(ctx, meta.Input, tools)
	plannerStep := r.startStep(run, 1, StepKindPlanner, "plan", meta.Input, "")
	plannerStep.Token = token
	if err != nil {
		plannerStep.Status = string(StepStatusFailed)
		plannerStep.Error = err.Error()
		r.finishStep(run, plannerStep, rawPlan)
		return r.failRun(run, err)
	}
	plannerStep.Status = string(StepStatusSucceeded)
	r.finishStep(run, plannerStep, rawPlan)

	if len(plans) == 0 {
		return r.finishDirectRun(ctx, run, meta.Input)
	}

	lastPlan := rawPlan
	evidence := make([]StepEvidence, 0)
	stepIndex := 2

	for loop := 0; loop < r.maxSteps(); loop++ {
		for _, plan := range plans {
			entry, ok := r.Registry.Get(plan.Name)
			if !ok {
				return r.failRun(run, fmt.Errorf("tool %s not found", plan.Name))
			}

			execStep := r.startStep(run, stepIndex, StepKindExecutor, plan.Name, plan.Description, entry.Spec.Name)
			stepIndex++

			result, stepToken, execErr := r.Executor.ExecuteStep(ctx, plan.Description, entry)
			execStep.Token = stepToken
			if execErr != nil {
				execStep.Status = string(StepStatusFailed)
				execStep.Error = execErr.Error()
				if result != nil {
					execStep.Observations = result.Observations
				}
				r.finishStep(run, execStep, toolOutput(result))
				return r.failRun(run, execErr)
			}

			execStep.Status = string(StepStatusSucceeded)
			execStep.Observations = result.Observations
			r.finishStep(run, execStep, toolOutput(result))

			evidence = append(evidence, StepEvidence{
				StepIndex:    execStep.StepIndex,
				Name:         plan.Name,
				ToolName:     entry.Spec.Name,
				Prompt:       plan.Description,
				Output:       result.Output,
				Observations: result.Observations,
			})
		}

		judgeStep := r.startStep(run, stepIndex, StepKindJudge, "judge", meta.Input, "")
		stepIndex++

		nextPlans, judgeRaw, judgeToken, judgeErr := r.Planner.JudgeTask(ctx, meta.Input, evidence, lastPlan, tools)
		judgeStep.Token = judgeToken
		if judgeErr != nil {
			judgeStep.Status = string(StepStatusFailed)
			judgeStep.Error = judgeErr.Error()
			r.finishStep(run, judgeStep, judgeRaw)
			return r.failRun(run, judgeErr)
		}

		judgeStep.Status = string(StepStatusSucceeded)
		r.finishStep(run, judgeStep, judgeRaw)
		if len(nextPlans) == 0 {
			break
		}

		plans = nextPlans
		lastPlan = judgeRaw
	}

	return r.finishSynthesizedRun(ctx, run, meta.Input, stepIndex, evidence)
}

func (r *Runner) RunMCP(ctx context.Context, meta RunMeta) (*db.AgentRun, error) {
	run := r.startRun(meta)
	tools := r.Registry.List()

	toolName, rawSelection, token, err := r.Planner.SelectTool(ctx, meta.Input, tools)
	plannerStep := r.startStep(run, 1, StepKindPlanner, "select_tool", meta.Input, "")
	plannerStep.Token = token
	if err != nil {
		plannerStep.Status = string(StepStatusFailed)
		plannerStep.Error = err.Error()
		r.finishStep(run, plannerStep, rawSelection)
		return r.failRun(run, err)
	}
	plannerStep.Status = string(StepStatusSucceeded)
	r.finishStep(run, plannerStep, rawSelection)

	if toolName == "" {
		return r.finishDirectRun(ctx, run, meta.Input)
	}

	entry, ok := r.Registry.Get(toolName)
	if !ok {
		return r.failRun(run, fmt.Errorf("tool %s not found", toolName))
	}

	execStep := r.startStep(run, 2, StepKindExecutor, toolName, meta.Input, entry.Spec.Name)
	result, stepToken, execErr := r.Executor.ExecuteDirect(ctx, meta.Input, entry)
	execStep.Token = stepToken
	if execErr != nil {
		execStep.Status = string(StepStatusFailed)
		execStep.Error = execErr.Error()
		if result != nil {
			execStep.Observations = result.Observations
		}
		r.finishStep(run, execStep, toolOutput(result))
		return r.failRun(run, execErr)
	}

	execStep.Status = string(StepStatusSucceeded)
	execStep.Observations = result.Observations
	r.finishStep(run, execStep, toolOutput(result))

	run.Status = string(RunStatusSucceeded)
	run.FinalOutput = result.Output
	r.updateRun(run)
	return run, nil
}

func (r *Runner) maxSteps() int {
	if r.MaxSteps <= 0 {
		return 6
	}
	return r.MaxSteps
}

func (r *Runner) finishDirectRun(ctx context.Context, run *db.AgentRun, input string) (*db.AgentRun, error) {
	output, token, err := r.Executor.RespondDirect(ctx, input)
	if err != nil {
		return r.failRun(run, err)
	}

	run.TokenTotal += token
	run.Status = string(RunStatusSucceeded)
	run.FinalOutput = output
	r.updateRun(run)
	return run, nil
}

func (r *Runner) finishSynthesizedRun(ctx context.Context, run *db.AgentRun, input string, stepIndex int, evidence []StepEvidence) (*db.AgentRun, error) {
	synthStep := r.startStep(run, stepIndex, StepKindSynthesizer, "synthesize", input, "")
	output, token, err := r.Executor.Synthesize(ctx, input, evidence)
	synthStep.Token = token
	if err != nil {
		synthStep.Status = string(StepStatusFailed)
		synthStep.Error = err.Error()
		r.finishStep(run, synthStep, output)
		return r.failRun(run, err)
	}

	synthStep.Status = string(StepStatusSucceeded)
	r.finishStep(run, synthStep, output)

	run.Status = string(RunStatusSucceeded)
	run.FinalOutput = output
	r.updateRun(run)
	return run, nil
}

func (r *Runner) failRun(run *db.AgentRun, err error) (*db.AgentRun, error) {
	run.Status = string(RunStatusFailed)
	run.Error = err.Error()
	r.updateRun(run)
	return run, err
}

func (r *Runner) startRun(meta RunMeta) *db.AgentRun {
	run := &db.AgentRun{
		UserId:   meta.UserID,
		ChatId:   meta.ChatID,
		MsgId:    meta.MsgID,
		Mode:     string(meta.Mode),
		Input:    meta.Input,
		ReplayOf: meta.ReplayOf,
		Status:   string(RunStatusRunning),
	}

	id, err := db.InsertAgentRun(run)
	if err != nil {
		logger.Error("insert agent run fail", "err", err)
		return run
	}
	run.ID = id
	return run
}

func (r *Runner) updateRun(run *db.AgentRun) {
	if run == nil || run.ID == 0 {
		return
	}
	if err := db.UpdateAgentRun(run); err != nil {
		logger.Error("update agent run fail", "err", err, "run_id", run.ID)
	}
}

func (r *Runner) startStep(run *db.AgentRun, stepIndex int, kind StepKind, name, input, toolName string) *db.AgentStep {
	provider, model := r.stepProviderModel()
	step := &db.AgentStep{
		RunID:     run.ID,
		StepIndex: stepIndex,
		Kind:      string(kind),
		Name:      name,
		Input:     input,
		Status:    string(StepStatusRunning),
		ToolName:  toolName,
		Provider:  provider,
		Model:     model,
	}

	id, err := db.InsertAgentStep(step)
	if err != nil {
		logger.Error("insert agent step fail", "err", err, "run_id", run.ID, "step_index", stepIndex)
		return step
	}
	step.ID = id
	return step
}

func (r *Runner) finishStep(run *db.AgentRun, step *db.AgentStep, output string) {
	if step == nil {
		return
	}
	step.RawOutput = output
	if run != nil {
		run.TokenTotal += step.Token
		if step.StepIndex > run.StepCount {
			run.StepCount = step.StepIndex
		}
		r.updateRun(run)
	}
	if step.ID == 0 {
		return
	}
	if err := db.UpdateAgentStep(step); err != nil {
		logger.Error("update agent step fail", "err", err, "step_id", step.ID)
	}
}

func (r *Runner) stepProviderModel() (string, string) {
	if provider, ok := r.Executor.(interface{ StepMetadata() (string, string) }); ok {
		return provider.StepMetadata()
	}
	if provider, ok := r.Planner.(interface{ StepMetadata() (string, string) }); ok {
		return provider.StepMetadata()
	}
	return "", ""
}

func toolOutput(result *tooling.ToolResult) string {
	if result == nil {
		return ""
	}
	return result.Output
}

func MarshalPlans(plans []TaskPlan) string {
	body, err := json.Marshal(plans)
	if err != nil {
		return ""
	}
	return string(body)
}

func NewObservation(function string, arguments map[string]interface{}, output string, err error) tooling.Observation {
	obs := tooling.Observation{
		Function:  function,
		Arguments: arguments,
		Output:    output,
		CreatedAt: time.Now().Unix(),
	}
	if err != nil {
		obs.Error = err.Error()
	}
	return obs
}
