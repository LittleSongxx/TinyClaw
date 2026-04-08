package agent

import (
	"context"
	"errors"
	"time"

	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/LittleSongxx/TinyClaw/session"
	"github.com/LittleSongxx/TinyClaw/tooling"
	"github.com/google/uuid"
)

type RunMode string

const (
	RunModeChat  RunMode = "chat"
	RunModeTask  RunMode = "task"
	RunModeSkill RunMode = "skill"
	RunModeMCP   RunMode = "mcp"
)

type ContextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Session  *session.Envelope  `json:"session,omitempty"`
	Input    string             `json:"input"`
	Mode     RunMode            `json:"mode"`
	Metadata map[string]string  `json:"metadata,omitempty"`
	Tools    []tooling.ToolSpec `json:"tools,omitempty"`
}

type Step struct {
	Index       int               `json:"index"`
	Kind        string            `json:"kind"`
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	Output      string            `json:"output,omitempty"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	StartedAt   int64             `json:"started_at"`
	CompletedAt int64             `json:"completed_at"`
}

type Run struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id,omitempty"`
	Input       string `json:"input"`
	Mode        string `json:"mode"`
	Status      string `json:"status"`
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
	StartedAt   int64  `json:"started_at"`
	CompletedAt int64  `json:"completed_at"`
}

type Result struct {
	Output string `json:"output"`
}

type ContextAssembler interface {
	Assemble(ctx context.Context, req Request) ([]ContextMessage, error)
}

type Executor interface {
	Execute(ctx context.Context, req Request, contextMessages []ContextMessage) (*Result, []Step, error)
}

type Runtime struct {
	Assembler ContextAssembler
	Executor  Executor
	Tools     *tooling.Broker
	Nodes     node.Broker
}

func NewRuntime(assembler ContextAssembler, executor Executor, tools *tooling.Broker, nodes node.Broker) *Runtime {
	return &Runtime{
		Assembler: assembler,
		Executor:  executor,
		Tools:     tools,
		Nodes:     nodes,
	}
}

func (r *Runtime) Run(ctx context.Context, req Request) (*Run, []Step, error) {
	run := &Run{
		ID:        uuid.NewString(),
		Input:     req.Input,
		Mode:      string(req.Mode),
		Status:    "running",
		StartedAt: time.Now().Unix(),
	}
	if req.Session != nil {
		run.SessionID = req.Session.SessionID
	}

	if r == nil || r.Executor == nil {
		run.Status = "failed"
		run.Error = "agent executor is not configured"
		run.CompletedAt = time.Now().Unix()
		return run, nil, errors.New(run.Error)
	}

	var (
		contextMessages []ContextMessage
		err             error
	)
	if r.Assembler != nil {
		contextMessages, err = r.Assembler.Assemble(ctx, req)
		if err != nil {
			run.Status = "failed"
			run.Error = err.Error()
			run.CompletedAt = time.Now().Unix()
			return run, nil, err
		}
	}

	result, steps, err := r.Executor.Execute(ctx, req, contextMessages)
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		run.CompletedAt = time.Now().Unix()
		return run, steps, err
	}

	run.Status = "completed"
	run.Output = result.Output
	run.CompletedAt = time.Now().Unix()
	return run, steps, nil
}

type SessionAssembler struct {
	Store session.Store
	Limit int
}

func (a *SessionAssembler) Assemble(ctx context.Context, req Request) ([]ContextMessage, error) {
	if a == nil || a.Store == nil || req.Session == nil {
		return nil, nil
	}
	limit := a.Limit
	if limit <= 0 {
		limit = 20
	}
	messages, err := a.Store.Recent(ctx, req.Session.SessionID, limit)
	if err != nil {
		return nil, err
	}

	out := make([]ContextMessage, 0, len(messages))
	for _, message := range messages {
		out = append(out, ContextMessage{
			Role:    message.Role,
			Content: message.Content,
		})
	}
	return out, nil
}
