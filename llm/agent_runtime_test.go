package llm

import (
	"testing"

	"github.com/LittleSongxx/TinyClaw/agentruntime"
	"github.com/stretchr/testify/assert"
)

func TestParseTaskPlans(t *testing.T) {
	plans, err := parseTaskPlans("plan:\n```json\n{\"plan\":[{\"name\":\"browser\",\"description\":\"collect docs\"}]}\n```")
	assert.NoError(t, err)
	if assert.Len(t, plans, 1) {
		assert.Equal(t, agentruntime.TaskPlan{
			Name:        "browser",
			Description: "collect docs",
		}, plans[0])
	}
}

func TestParseSelectedTool(t *testing.T) {
	toolName, err := parseSelectedTool("result: {\"agent\":\"browser\"}")
	assert.NoError(t, err)
	assert.Equal(t, "browser", toolName)
}
