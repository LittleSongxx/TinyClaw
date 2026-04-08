package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecMcpReqRejectsDisallowedTool(t *testing.T) {
	client := &LLM{
		AllowedTools: map[string]bool{
			"allowed_tool": true,
		},
	}

	_, err := client.ExecMcpReq(context.Background(), "blocked_tool", map[string]interface{}{
		"path": "/tmp/demo",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}
