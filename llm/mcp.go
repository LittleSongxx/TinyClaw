package llm

import (
	"github.com/LittleSongxx/TinyClaw/agentruntime"
	"github.com/LittleSongxx/TinyClaw/db"
)

// ExecuteMcp execute mcp request
func (d *LLMTaskReq) ExecuteMcp() error {
	_, err := d.ExecuteMcpRun()
	return err
}

func (d *LLMTaskReq) ExecuteMcpRun() (*db.AgentRun, error) {
	runner := d.newRunner()
	return runner.RunMCP(d.runtimeContext(), d.runMeta(agentruntime.ModeMCP))
}
