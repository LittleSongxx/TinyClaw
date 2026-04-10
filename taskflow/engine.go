package taskflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/LittleSongxx/TinyClaw/tooling"
	"github.com/google/uuid"
)

type NodeState struct {
	Status  string                 `json:"status"`
	Outputs map[string]interface{} `json:"outputs,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

type Handlers struct {
	Tool        func(context.Context, tooling.ToolInvocation) (*tooling.ToolResult, error)
	NodeCommand func(context.Context, node.NodeCommandRequest) (*node.NodeCommandResult, error)
	AgentRun    func(context.Context, map[string]interface{}) (map[string]interface{}, error)
	Subflow     func(context.Context, string, map[string]interface{}) (map[string]interface{}, error)
}

type Engine struct {
	handlers Handlers
}

func NewEngine(handlers Handlers) *Engine {
	return &Engine{handlers: handlers}
}

func CreateOrUpdate(ctx context.Context, flowID, name, description string, spec Spec) (*db.TaskFlowRecord, error) {
	principal, err := authz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if !principal.CanManageWorkspace() {
		return nil, authz.ErrForbidden
	}
	if err := Validate(spec); err != nil {
		return nil, err
	}
	if flowID == "" {
		flowID = uuid.NewString()
	}
	current, _ := db.GetTaskFlow(ctx, principal.WorkspaceID, flowID)
	version := 1
	if current != nil {
		version = current.CurrentVersion + 1
	}
	specMap, _ := specToMap(spec)
	record := db.TaskFlowRecord{
		FlowID:         flowID,
		WorkspaceID:    principal.WorkspaceID,
		Name:           name,
		Description:    description,
		CurrentVersion: version,
		Status:         "active",
	}
	if err := db.UpsertTaskFlow(ctx, record, specMap); err != nil {
		return nil, err
	}
	return db.GetTaskFlow(ctx, principal.WorkspaceID, flowID)
}

func ValidateStoredSpec(specMap map[string]interface{}) (Spec, error) {
	var spec Spec
	body, _ := json.Marshal(specMap)
	if err := json.Unmarshal(body, &spec); err != nil {
		return spec, err
	}
	return spec, Validate(spec)
}

func (e *Engine) Run(ctx context.Context, flowID string, inputs map[string]interface{}) (*db.TaskFlowRunRecord, error) {
	principal, err := authz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	versionRecord, err := db.GetTaskFlowVersion(ctx, principal.WorkspaceID, flowID, 0)
	if err != nil {
		return nil, err
	}
	if versionRecord == nil {
		return nil, errors.New("flow not found")
	}
	spec, err := ValidateStoredSpec(versionRecord.Spec)
	if err != nil {
		return nil, err
	}
	run := &db.TaskFlowRunRecord{
		RunID:       uuid.NewString(),
		WorkspaceID: principal.WorkspaceID,
		FlowID:      flowID,
		Version:     versionRecord.Version,
		Status:      StatusRunning,
		Inputs:      inputs,
		Outputs:     map[string]interface{}{},
	}
	if err := db.UpsertTaskFlowRun(ctx, *run); err != nil {
		return nil, err
	}
	_ = db.InsertTaskFlowEvent(ctx, db.TaskFlowEventRecord{WorkspaceID: principal.WorkspaceID, RunID: run.RunID, Event: "flow.started", Payload: inputs})
	e.execute(ctx, principal, run, spec)
	return db.GetTaskFlowRun(ctx, principal.WorkspaceID, run.RunID)
}

func (e *Engine) RetryNode(ctx context.Context, runID, nodeID string) (*db.TaskFlowRunRecord, error) {
	principal, err := authz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	run, err := db.GetTaskFlowRun(ctx, principal.WorkspaceID, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, errors.New("flow run not found")
	}
	versionRecord, err := db.GetTaskFlowVersion(ctx, principal.WorkspaceID, run.FlowID, run.Version)
	if err != nil {
		return nil, err
	}
	if versionRecord == nil {
		return nil, errors.New("flow version not found")
	}
	spec, err := ValidateStoredSpec(versionRecord.Spec)
	if err != nil {
		return nil, err
	}
	var target *NodeSpec
	for index := range spec.Nodes {
		if spec.Nodes[index].ID == nodeID {
			target = &spec.Nodes[index]
			break
		}
	}
	if target == nil {
		return nil, errors.New("flow node not found")
	}
	outputs, execErr := e.executeNode(authz.WithPrincipal(ctx, principal), run, *target)
	status := StatusSucceeded
	errText := ""
	if execErr != nil {
		status = StatusFailed
		errText = execErr.Error()
	}
	_ = db.UpsertTaskFlowNodeRun(ctx, db.TaskFlowNodeRunRecord{
		WorkspaceID: principal.WorkspaceID,
		RunID:       run.RunID,
		NodeID:      target.ID,
		NodeType:    target.Type,
		Status:      status,
		Outputs:     outputs,
		Error:       errText,
		Attempt:     1,
		CompletedAt: time.Now().Unix(),
	})
	if run.Outputs == nil {
		run.Outputs = map[string]interface{}{}
	}
	run.Outputs[target.ID] = NodeState{Status: status, Outputs: outputs, Error: errText}
	run.Status = status
	run.Error = errText
	if status == StatusSucceeded {
		run.Status = StatusRunning
	}
	if err := db.UpsertTaskFlowRun(ctx, *run); err != nil {
		return nil, err
	}
	if execErr != nil {
		return run, execErr
	}
	return db.GetTaskFlowRun(ctx, principal.WorkspaceID, run.RunID)
}

func (e *Engine) execute(ctx context.Context, principal authz.Principal, run *db.TaskFlowRunRecord, spec Spec) {
	state := make(map[string]NodeState, len(spec.Nodes))
	nodes := make(map[string]NodeSpec, len(spec.Nodes))
	deps := make(map[string][]EdgeSpec)
	for _, nodeSpec := range spec.Nodes {
		nodes[nodeSpec.ID] = nodeSpec
		state[nodeSpec.ID] = NodeState{Status: StatusPending}
		_ = db.UpsertTaskFlowNodeRun(ctx, db.TaskFlowNodeRunRecord{
			WorkspaceID: principal.WorkspaceID,
			RunID:       run.RunID,
			NodeID:      nodeSpec.ID,
			NodeType:    nodeSpec.Type,
			Status:      StatusPending,
		})
	}
	for _, edge := range spec.Edges {
		deps[edge.To] = append(deps[edge.To], edge)
	}
	maxConcurrency := spec.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 4
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency)
	progress := true
	failed := false
	for progress {
		progress = false
		for id, nodeSpec := range nodes {
			mu.Lock()
			current := state[id]
			if current.Status != StatusPending || !dependenciesReady(deps[id], state) {
				mu.Unlock()
				continue
			}
			if !edgeConditionsPass(deps[id], state) || !evalJSONLogic(nodeSpec.Condition, state) {
				state[id] = NodeState{Status: StatusSkipped}
				mu.Unlock()
				_ = db.UpsertTaskFlowNodeRun(ctx, db.TaskFlowNodeRunRecord{WorkspaceID: principal.WorkspaceID, RunID: run.RunID, NodeID: id, NodeType: nodeSpec.Type, Status: StatusSkipped, CompletedAt: time.Now().Unix()})
				progress = true
				continue
			}
			state[id] = NodeState{Status: StatusRunning}
			mu.Unlock()
			progress = true
			wg.Add(1)
			sem <- struct{}{}
			go func(current NodeSpec) {
				defer wg.Done()
				defer func() { <-sem }()
				outputs, err := e.executeNode(authz.WithPrincipal(ctx, principal), run, current)
				status := StatusSucceeded
				errText := ""
				if err != nil {
					status = StatusFailed
					errText = err.Error()
				}
				mu.Lock()
				state[current.ID] = NodeState{Status: status, Outputs: outputs, Error: errText}
				if status == StatusFailed {
					failed = true
				}
				mu.Unlock()
				_ = db.UpsertTaskFlowNodeRun(ctx, db.TaskFlowNodeRunRecord{
					WorkspaceID: principal.WorkspaceID,
					RunID:       run.RunID,
					NodeID:      current.ID,
					NodeType:    current.Type,
					Status:      status,
					Outputs:     outputs,
					Error:       errText,
					Attempt:     1,
					CompletedAt: time.Now().Unix(),
				})
			}(nodeSpec)
		}
		wg.Wait()
	}

	outputs := map[string]interface{}{}
	for id, item := range state {
		outputs[id] = item
	}
	run.Outputs = outputs
	run.CompletedAt = time.Now().Unix()
	if failed || hasUnfinished(state) {
		run.Status = StatusFailed
		run.Error = "one or more flow nodes failed or remained pending"
	} else {
		run.Status = StatusSucceeded
	}
	_ = db.UpsertTaskFlowRun(ctx, *run)
	_ = db.InsertTaskFlowEvent(ctx, db.TaskFlowEventRecord{WorkspaceID: principal.WorkspaceID, RunID: run.RunID, Event: "flow." + run.Status, Payload: outputs})
}

func (e *Engine) executeNode(ctx context.Context, run *db.TaskFlowRunRecord, spec NodeSpec) (map[string]interface{}, error) {
	_ = db.UpsertTaskFlowNodeRun(ctx, db.TaskFlowNodeRunRecord{
		WorkspaceID: run.WorkspaceID,
		RunID:       run.RunID,
		NodeID:      spec.ID,
		NodeType:    spec.Type,
		Status:      StatusRunning,
		Inputs:      spec.Params,
		Attempt:     1,
	})
	_ = db.InsertTaskFlowEvent(ctx, db.TaskFlowEventRecord{WorkspaceID: run.WorkspaceID, RunID: run.RunID, NodeID: spec.ID, Event: "node.started", Payload: spec.Params})
	switch spec.Type {
	case NodeTypeCondition:
		return map[string]interface{}{"matched": true}, nil
	case NodeTypeApproval:
		return nil, errors.New("approval node is waiting for external decision")
	case NodeTypeTool:
		if e.handlers.Tool == nil {
			return nil, errors.New("tool handler is not configured")
		}
		result, err := e.handlers.Tool(ctx, tooling.ToolInvocation{Name: spec.Tool, Arguments: spec.Params})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"output": result.Output, "error": result.Error}, nil
	case NodeTypeNodeCommand:
		if e.handlers.NodeCommand == nil {
			return nil, errors.New("node command handler is not configured")
		}
		req := node.NodeCommandRequest{Capability: fmt.Sprint(spec.Params["capability"]), Arguments: spec.Params}
		if nodeID, ok := spec.Params["node_id"].(string); ok {
			req.NodeID = nodeID
		}
		result, err := e.handlers.NodeCommand(ctx, req)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"success": result.Success, "output": result.Output, "error": result.Error}, nil
	case NodeTypeAgentRun:
		if e.handlers.AgentRun == nil {
			return nil, errors.New("agent_run handler is not configured")
		}
		return e.handlers.AgentRun(ctx, spec.Params)
	case NodeTypeSubflow:
		if e.handlers.Subflow == nil {
			return nil, errors.New("subflow handler is not configured")
		}
		return e.handlers.Subflow(ctx, spec.FlowID, spec.Params)
	default:
		return nil, errors.New("unsupported node type: " + spec.Type)
	}
}

func dependenciesReady(edges []EdgeSpec, state map[string]NodeState) bool {
	for _, edge := range edges {
		status := state[edge.From].Status
		if status != StatusSucceeded && status != StatusSkipped && status != StatusFailed {
			return false
		}
	}
	return true
}

func edgeConditionsPass(edges []EdgeSpec, state map[string]NodeState) bool {
	for _, edge := range edges {
		if state[edge.From].Status == StatusFailed {
			return false
		}
		if !evalJSONLogic(edge.Condition, state) {
			return false
		}
	}
	return true
}

func hasUnfinished(state map[string]NodeState) bool {
	for _, item := range state {
		if item.Status == StatusPending || item.Status == StatusRunning || item.Status == StatusReady || item.Status == StatusWaiting {
			return true
		}
	}
	return false
}

func specToMap(spec Spec) (map[string]interface{}, error) {
	body, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func validateAcyclic(spec Spec) error {
	inDegree := make(map[string]int, len(spec.Nodes))
	children := make(map[string][]string)
	for _, node := range spec.Nodes {
		inDegree[node.ID] = 0
	}
	for _, edge := range spec.Edges {
		inDegree[edge.To]++
		children[edge.From] = append(children[edge.From], edge.To)
	}
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, child := range children[id] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}
	if visited != len(inDegree) {
		return errors.New("flow DAG contains a cycle")
	}
	return nil
}
