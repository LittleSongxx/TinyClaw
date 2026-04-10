package taskflow

import (
	"errors"
	"strings"
	"time"
)

const (
	NodeTypeAgentRun    = "agent_run"
	NodeTypeTool        = "tool"
	NodeTypeNodeCommand = "node_command"
	NodeTypeApproval    = "approval"
	NodeTypeCondition   = "condition"
	NodeTypeSubflow     = "subflow"

	StatusPending   = "pending"
	StatusReady     = "ready"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusSkipped   = "skipped"
	StatusCancelled = "cancelled"
	StatusWaiting   = "waiting"
)

type Spec struct {
	Nodes          []NodeSpec              `json:"nodes"`
	Edges          []EdgeSpec              `json:"edges,omitempty"`
	Inputs         map[string]interface{}  `json:"inputs,omitempty"`
	Outputs        map[string]interface{}  `json:"outputs,omitempty"`
	MaxConcurrency int                     `json:"max_concurrency,omitempty"`
	Timeout        string                  `json:"timeout,omitempty"`
	RetryPolicy    RetryPolicy             `json:"retry_policy,omitempty"`
}

type NodeSpec struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name,omitempty"`
	Tool        string                 `json:"tool,omitempty"`
	FlowID      string                 `json:"flow_id,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Condition   map[string]interface{} `json:"condition,omitempty"`
	RetryPolicy *RetryPolicy           `json:"retry_policy,omitempty"`
	Timeout     string                 `json:"timeout,omitempty"`
}

type EdgeSpec struct {
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Condition map[string]interface{} `json:"condition,omitempty"`
}

type RetryPolicy struct {
	MaxAttempts int    `json:"max_attempts,omitempty"`
	Backoff     string `json:"backoff,omitempty"`
}

func Validate(spec Spec) error {
	if len(spec.Nodes) == 0 {
		return errors.New("flow nodes are required")
	}
	seen := make(map[string]NodeSpec, len(spec.Nodes))
	for _, node := range spec.Nodes {
		node.ID = strings.TrimSpace(node.ID)
		if node.ID == "" {
			return errors.New("node id is required")
		}
		if _, ok := seen[node.ID]; ok {
			return errors.New("duplicate node id: " + node.ID)
		}
		switch node.Type {
		case NodeTypeAgentRun, NodeTypeTool, NodeTypeNodeCommand, NodeTypeApproval, NodeTypeCondition, NodeTypeSubflow:
		default:
			return errors.New("unsupported node type: " + node.Type)
		}
		if _, err := parseOptionalDuration(node.Timeout); err != nil {
			return err
		}
		if err := validateJSONLogic(node.Condition); err != nil {
			return err
		}
		seen[node.ID] = node
	}
	for _, edge := range spec.Edges {
		if _, ok := seen[edge.From]; !ok {
			return errors.New("edge from node not found: " + edge.From)
		}
		if _, ok := seen[edge.To]; !ok {
			return errors.New("edge to node not found: " + edge.To)
		}
		if err := validateJSONLogic(edge.Condition); err != nil {
			return err
		}
	}
	return validateAcyclic(spec)
}

func parseOptionalDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	return time.ParseDuration(value)
}
