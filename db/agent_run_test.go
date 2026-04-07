package db

import (
	"testing"

	"github.com/LittleSongxx/TinyClaw/tooling"
	"github.com/stretchr/testify/assert"
)

func TestInsertAndGetAgentRunDetail(t *testing.T) {
	run := &AgentRun{
		UserId: "agent-user",
		ChatId: "chat-1",
		MsgId:  "msg-1",
		Mode:   "task",
		Input:  "find and summarize docs",
		Status: "running",
	}

	runID, err := InsertAgentRun(run)
	assert.NoError(t, err)
	assert.NotZero(t, runID)

	step := &AgentStep{
		RunID:     runID,
		StepIndex: 1,
		Kind:      "planner",
		Name:      "plan",
		Input:     run.Input,
		Status:    "succeeded",
		RawOutput: `{"plan":[]}`,
		Token:     42,
		Observations: []tooling.Observation{
			{
				Function:  "search",
				Output:    "result body",
				CreatedAt: 123,
			},
		},
	}

	stepID, err := InsertAgentStep(step)
	assert.NoError(t, err)
	assert.NotZero(t, stepID)

	run.ID = runID
	run.Status = "succeeded"
	run.FinalOutput = "all done"
	assert.NoError(t, UpdateAgentRun(run))

	step.ID = stepID
	step.RawOutput = "updated output"
	assert.NoError(t, UpdateAgentStep(step))

	detail, err := GetAgentRunDetailByID(runID)
	assert.NoError(t, err)
	if assert.NotNil(t, detail) && assert.NotNil(t, detail.Run) {
		assert.Equal(t, "succeeded", detail.Run.Status)
		assert.Equal(t, "all done", detail.Run.FinalOutput)
	}

	if assert.Len(t, detail.Steps, 1) {
		assert.Equal(t, "updated output", detail.Steps[0].RawOutput)
		assert.Equal(t, 1, len(detail.Steps[0].Observations))
		assert.Equal(t, "search", detail.Steps[0].Observations[0].Function)
	}
}
