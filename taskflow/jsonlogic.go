package taskflow

import (
	"errors"
	"fmt"
	"strings"
)

var allowedJSONLogicOps = map[string]bool{
	"and":       true,
	"or":        true,
	"not":       true,
	"==":        true,
	"!=":        true,
	"contains":  true,
	"exists":    true,
	"status_is": true,
}

func validateJSONLogic(expr map[string]interface{}) error {
	if len(expr) == 0 {
		return nil
	}
	if len(expr) != 1 {
		return errors.New("jsonlogic expression must contain exactly one operator")
	}
	for op, value := range expr {
		if !allowedJSONLogicOps[op] {
			return fmt.Errorf("jsonlogic operator %q is not allowed", op)
		}
		switch typed := value.(type) {
		case []interface{}:
			for _, item := range typed {
				if nested, ok := item.(map[string]interface{}); ok {
					if err := validateJSONLogic(nested); err != nil {
						return err
					}
				}
			}
		case map[string]interface{}:
			return validateJSONLogic(typed)
		}
	}
	return nil
}

func evalJSONLogic(expr map[string]interface{}, state map[string]NodeState) bool {
	if len(expr) == 0 {
		return true
	}
	for op, value := range expr {
		switch op {
		case "and":
			for _, item := range asList(value) {
				if !evalAny(item, state) {
					return false
				}
			}
			return true
		case "or":
			for _, item := range asList(value) {
				if evalAny(item, state) {
					return true
				}
			}
			return false
		case "not":
			return !evalAny(value, state)
		case "==":
			items := asList(value)
			return len(items) >= 2 && resolveValue(items[0], state) == resolveValue(items[1], state)
		case "!=":
			items := asList(value)
			return len(items) >= 2 && resolveValue(items[0], state) != resolveValue(items[1], state)
		case "contains":
			items := asList(value)
			if len(items) < 2 {
				return false
			}
			return strings.Contains(resolveValue(items[0], state), resolveValue(items[1], state))
		case "exists":
			return resolveValue(value, state) != ""
		case "status_is":
			items := asList(value)
			if len(items) < 2 {
				return false
			}
			nodeID := resolveValue(items[0], state)
			return state[nodeID].Status == resolveValue(items[1], state)
		}
	}
	return false
}

func evalAny(value interface{}, state map[string]NodeState) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case map[string]interface{}:
		return evalJSONLogic(typed, state)
	case string:
		return resolveValue(typed, state) != ""
	default:
		return typed != nil
	}
}

func resolveValue(value interface{}, state map[string]NodeState) string {
	text, ok := value.(string)
	if !ok {
		return fmt.Sprint(value)
	}
	if !strings.HasPrefix(text, "$.") {
		return text
	}
	path := strings.Split(strings.TrimPrefix(text, "$."), ".")
	if len(path) < 2 {
		return ""
	}
	nodeState := state[path[0]]
	switch path[1] {
	case "status":
		return nodeState.Status
	case "output", "outputs":
		if len(path) >= 3 {
			return fmt.Sprint(nodeState.Outputs[path[2]])
		}
		return fmt.Sprint(nodeState.Outputs)
	default:
		return ""
	}
}

func asList(value interface{}) []interface{} {
	if value == nil {
		return nil
	}
	if items, ok := value.([]interface{}); ok {
		return items
	}
	return []interface{}{value}
}
