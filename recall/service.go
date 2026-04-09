package recall

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/tooling"
	"github.com/LittleSongxx/mcp-client-go/clients"
	"github.com/mark3labs/mcp-go/mcp"
)

type Service struct{}

var (
	defaultService *Service
	once           sync.Once
)

func DefaultService() *Service {
	once.Do(func() {
		defaultService = &Service{}
	})
	return defaultService
}

func (s *Service) BuildConversationContext(userID string, maxPairs int) string {
	if strings.TrimSpace(userID) == "" || maxPairs <= 0 {
		return ""
	}

	record := db.GetMsgRecord(userID)
	if record == nil || len(record.AQs) == 0 {
		return ""
	}

	start := len(record.AQs) - maxPairs
	if start < 0 {
		start = 0
	}

	lines := make([]string, 0, maxPairs)
	for _, item := range record.AQs[start:] {
		question := strings.TrimSpace(item.Question)
		answer := strings.TrimSpace(item.Answer)
		if question == "" && answer == "" {
			continue
		}

		lines = append(lines, fmt.Sprintf("User: %s\nAssistant: %s",
			trimContext(question, 240), trimContext(answer, 240)))
	}
	return strings.Join(lines, "\n\n")
}

func (s *Service) SearchLongTermMemory(ctx context.Context, userID, userTask, skillID string) (string, tooling.Observation) {
	obs := tooling.Observation{
		Function:  "memory_search",
		Arguments: map[string]interface{}{"query": userTask, "skill_id": skillID, "user_id": userID},
		CreatedAt: time.Now().Unix(),
	}

	memoryClient, err := clients.GetMCPClient("memory")
	if err != nil {
		return "", tooling.Observation{}
	}

	toolName := ""
	args := map[string]interface{}{}
	entityName := memoryEntityName(userID, skillID)

	if hasTool(memoryClient.Tools, "search_nodes") {
		toolName = "search_nodes"
		args["query"] = userTask
	} else if tool, ok := findToolByKeyword(memoryClient.Tools, "search", "query", "find"); ok {
		toolName = tool.Name
		args = buildGenericArgs(tool, userTask, "", entityName)
	} else if hasTool(memoryClient.Tools, "read_graph") {
		toolName = "read_graph"
	} else {
		return "", tooling.Observation{}
	}

	output, err := memoryClient.ExecTools(ctx, toolName, args)
	obs.Function = toolName
	obs.Arguments = args
	if err != nil {
		obs.Error = err.Error()
		return "", obs
	}

	return trimContext(output, 1800), obs
}

func (s *Service) StoreLongTermMemory(ctx context.Context, userID, skillID, userTask, output string) tooling.Observation {
	summary := trimContext(fmt.Sprintf("User task: %s\nSkill: %s\nOutcome: %s", userTask, skillID, output), 1200)
	obs := tooling.Observation{
		Function:  "memory_store",
		Arguments: map[string]interface{}{"skill_id": skillID, "user_id": userID},
		Output:    summary,
		CreatedAt: time.Now().Unix(),
	}

	memoryClient, err := clients.GetMCPClient("memory")
	if err != nil {
		return tooling.Observation{}
	}

	entityName := memoryEntityName(userID, skillID)
	toolName := ""
	args := map[string]interface{}{}

	switch {
	case hasTool(memoryClient.Tools, "add_observations"):
		toolName = "add_observations"
		args = map[string]interface{}{
			"observations": []map[string]interface{}{
				{
					"entityName": entityName,
					"contents":   []string{summary},
				},
			},
		}
	case hasTool(memoryClient.Tools, "create_entities"):
		toolName = "create_entities"
		args = map[string]interface{}{
			"entities": []map[string]interface{}{
				{
					"name":         entityName,
					"entityType":   "skill_memory",
					"observations": []string{summary},
				},
			},
		}
	default:
		tool, ok := findToolByKeyword(memoryClient.Tools, "observation", "entity", "store", "append", "create")
		if !ok {
			return tooling.Observation{}
		}
		toolName = tool.Name
		args = buildGenericArgs(tool, userTask, summary, entityName)
	}

	resp, err := memoryClient.ExecTools(ctx, toolName, args)
	obs.Function = toolName
	obs.Arguments = args
	if err != nil {
		obs.Error = err.Error()
		return obs
	}
	obs.Output = trimContext(resp, 800)
	return obs
}

func (s *Service) MemoryStatus(ctx context.Context) MemoryStatus {
	status := MemoryStatus{
		Enabled:  true,
		Provider: "mcp:memory",
		Mode:     "conversation+long_term",
	}

	client, err := clients.GetMCPClient("memory")
	if err != nil {
		status.LastError = err.Error()
		return status
	}

	status.Available = true
	status.Tools = make([]string, 0, len(client.Tools))
	for _, tool := range client.Tools {
		status.Tools = append(status.Tools, tool.Name)
		if tool.Name == "search_nodes" || tool.Name == "read_graph" || strings.Contains(strings.ToLower(tool.Name), "graph") {
			status.HasGraphStore = true
		}
	}
	return status
}

func hasTool(tools []mcp.Tool, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func findToolByKeyword(tools []mcp.Tool, keywords ...string) (mcp.Tool, bool) {
	for _, tool := range tools {
		name := strings.ToLower(tool.Name)
		for _, keyword := range keywords {
			if strings.Contains(name, strings.ToLower(keyword)) {
				return tool, true
			}
		}
	}
	return mcp.Tool{}, false
}

func buildGenericArgs(tool mcp.Tool, query, summary, entityName string) map[string]interface{} {
	args := make(map[string]interface{})
	for name, prop := range tool.InputSchema.Properties {
		propSchema, _ := prop.(map[string]interface{})
		propType, _ := propSchema["type"].(string)
		lowerName := strings.ToLower(name)

		switch propType {
		case "string":
			switch {
			case strings.Contains(lowerName, "query"), strings.Contains(lowerName, "search"), strings.Contains(lowerName, "prompt"),
				strings.Contains(lowerName, "input"), strings.Contains(lowerName, "text"):
				args[name] = query
			case strings.Contains(lowerName, "entity"), strings.Contains(lowerName, "name"), strings.Contains(lowerName, "node"):
				args[name] = entityName
			case strings.Contains(lowerName, "summary"), strings.Contains(lowerName, "content"), strings.Contains(lowerName, "description"):
				args[name] = summary
			default:
				args[name] = summary
			}
		case "integer", "number":
			if strings.Contains(lowerName, "limit") || strings.Contains(lowerName, "count") || strings.Contains(lowerName, "max") {
				args[name] = 5
			}
		case "boolean":
			args[name] = true
		case "array":
			switch {
			case strings.Contains(lowerName, "content"), strings.Contains(lowerName, "observation"):
				args[name] = []string{summary}
			case strings.Contains(lowerName, "name"), strings.Contains(lowerName, "entity"), strings.Contains(lowerName, "node"):
				args[name] = []string{entityName}
			}
		}
	}
	return args
}

func memoryEntityName(userID, skillID string) string {
	name := fmt.Sprintf("tinyclaw_%s_%s", skillID, userID)
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.Trim(name, "_")
	if name == "" {
		name = "tinyclaw_skill_memory"
	}
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

func trimContext(content string, limit int) string {
	content = strings.TrimSpace(content)
	if limit <= 0 || len(content) <= limit {
		return content
	}
	return strings.TrimSpace(content[:limit]) + "..."
}
