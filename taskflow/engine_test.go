package taskflow

import (
	"testing"
)

func TestValidateRejectsCycles(t *testing.T) {
	spec := Spec{
		Nodes: []NodeSpec{
			{ID: "a", Type: NodeTypeTool},
			{ID: "b", Type: NodeTypeTool},
		},
		Edges: []EdgeSpec{
			{From: "a", To: "b"},
			{From: "b", To: "a"},
		},
	}
	if err := Validate(spec); err == nil {
		t.Fatal("expected cycle validation error")
	}
}

func TestJSONLogicSubsetReadsOnlyNodeState(t *testing.T) {
	state := map[string]NodeState{
		"build": {Status: StatusSucceeded, Outputs: map[string]interface{}{"artifact": "tinyclaw"}},
	}
	expr := map[string]interface{}{
		"and": []interface{}{
			map[string]interface{}{"status_is": []interface{}{"build", StatusSucceeded}},
			map[string]interface{}{"contains": []interface{}{"$.build.outputs.artifact", "claw"}},
		},
	}
	if err := validateJSONLogic(expr); err != nil {
		t.Fatalf("validate jsonlogic: %v", err)
	}
	if !evalJSONLogic(expr, state) {
		t.Fatalf("expected expression to match")
	}
}

func TestValidateRejectsUnsafeJSONLogicOperator(t *testing.T) {
	err := Validate(Spec{
		Nodes: []NodeSpec{{
			ID:        "condition",
			Type:      NodeTypeCondition,
			Condition: map[string]interface{}{"script": "os.exit(1)"},
		}},
	})
	if err == nil {
		t.Fatal("expected unsafe jsonlogic operator rejection")
	}
}
