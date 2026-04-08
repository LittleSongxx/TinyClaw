package skill

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/tooling"
	"github.com/LittleSongxx/mcp-client-go/clients"
	"github.com/mark3labs/mcp-go/mcp"
)

type MemoryContext struct {
	Conversation string `json:"conversation,omitempty"`
	LongTerm     string `json:"long_term,omitempty"`
}

func BuildPromptWithMemory(entry *tooling.Entry, userTask string, memory MemoryContext) string {
	if entry == nil || entry.Skill == nil {
		return userTask
	}

	skillInfo := entry.Skill
	var builder strings.Builder
	builder.WriteString("You are executing a dedicated skill.\n\n")
	builder.WriteString("Skill ID: ")
	builder.WriteString(skillInfo.ID)
	builder.WriteString("\nSkill Name: ")
	builder.WriteString(skillInfo.Name)
	builder.WriteString("\nSkill Description: ")
	builder.WriteString(skillInfo.Description)
	builder.WriteString("\nMemory Mode: ")
	builder.WriteString(skillInfo.Memory)
	if strings.TrimSpace(skillInfo.WhenToUse) != "" {
		builder.WriteString("\n\nWhen To Use:\n")
		builder.WriteString(skillInfo.WhenToUse)
	}
	if strings.TrimSpace(skillInfo.WhenNotToUse) != "" {
		builder.WriteString("\n\nWhen Not To Use:\n")
		builder.WriteString(skillInfo.WhenNotToUse)
	}
	if strings.TrimSpace(skillInfo.Instructions) != "" {
		builder.WriteString("\n\nInstructions:\n")
		builder.WriteString(skillInfo.Instructions)
	}
	if strings.TrimSpace(skillInfo.OutputContract) != "" {
		builder.WriteString("\n\nOutput Contract:\n")
		builder.WriteString(skillInfo.OutputContract)
	}
	if strings.TrimSpace(skillInfo.FailureHandling) != "" {
		builder.WriteString("\n\nFailure Handling:\n")
		builder.WriteString(skillInfo.FailureHandling)
	}
	if strings.TrimSpace(memory.Conversation) != "" {
		builder.WriteString("\n\nConversation Memory:\n")
		builder.WriteString(memory.Conversation)
	}
	if strings.TrimSpace(memory.LongTerm) != "" {
		builder.WriteString("\n\nLong-Term Memory:\n")
		builder.WriteString(memory.LongTerm)
	}
	if len(skillInfo.AllowedTools) > 0 {
		builder.WriteString("\n\nAllowed Tools:\n")
		for _, toolName := range skillInfo.AllowedTools {
			builder.WriteString("- ")
			builder.WriteString(toolName)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\nCurrent User Task:\n")
	builder.WriteString(userTask)
	return builder.String()
}

func LoadMemoryContext(ctx context.Context, userID, userTask string, entry *tooling.Entry) (MemoryContext, []tooling.Observation) {
	result := MemoryContext{}
	observations := make([]tooling.Observation, 0)
	if entry == nil || entry.Skill == nil {
		return result, observations
	}

	switch entry.Skill.Memory {
	case MemoryConversation, MemoryBoth:
		result.Conversation = BuildConversationContext(userID, 4)
		if result.Conversation != "" {
			observations = append(observations, tooling.Observation{
				Function:  "skill_conversation_memory",
				Output:    result.Conversation,
				CreatedAt: time.Now().Unix(),
			})
		}
	}

	switch entry.Skill.Memory {
	case MemoryLongTerm, MemoryBoth:
		longTerm, obs := searchLongTermMemory(ctx, userTask, entry)
		if strings.TrimSpace(longTerm) != "" {
			result.LongTerm = longTerm
		}
		if obs.Function != "" {
			observations = append(observations, obs)
		}
	}

	return result, observations
}

func PersistMemoryContext(ctx context.Context, userID, userTask, output string, entry *tooling.Entry) []tooling.Observation {
	if entry == nil || entry.Skill == nil {
		return nil
	}
	if entry.Skill.Memory != MemoryLongTerm && entry.Skill.Memory != MemoryBoth {
		return nil
	}

	observation := storeLongTermMemory(ctx, userID, userTask, output, entry)
	if observation.Function == "" {
		return nil
	}
	return []tooling.Observation{observation}
}

func BuildConversationContext(userID string, maxPairs int) string {
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

func searchLongTermMemory(ctx context.Context, userTask string, entry *tooling.Entry) (string, tooling.Observation) {
	obs := tooling.Observation{
		Function:  "memory_search",
		Arguments: map[string]interface{}{"query": userTask},
		CreatedAt: time.Now().Unix(),
	}

	memoryClient, err := clients.GetMCPClient("memory")
	if err != nil {
		return "", tooling.Observation{}
	}

	toolName := ""
	args := map[string]interface{}{}
	if hasTool(memoryClient.Tools, "search_nodes") {
		toolName = "search_nodes"
		args["query"] = userTask
	} else if tool, ok := findToolByKeyword(memoryClient.Tools, "search", "query", "find"); ok {
		toolName = tool.Name
		args = buildGenericArgs(tool, userTask, "", memoryEntityName("", entry.Skill.ID))
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

func storeLongTermMemory(ctx context.Context, userID, userTask, output string, entry *tooling.Entry) tooling.Observation {
	summary := trimContext(fmt.Sprintf("User task: %s\nSkill: %s\nOutcome: %s", userTask, entry.Skill.ID, output), 1200)
	obs := tooling.Observation{
		Function:  "memory_store",
		Arguments: map[string]interface{}{"skill_id": entry.Skill.ID},
		Output:    summary,
		CreatedAt: time.Now().Unix(),
	}

	memoryClient, err := clients.GetMCPClient("memory")
	if err != nil {
		return tooling.Observation{}
	}

	entityName := memoryEntityName(userID, entry.Skill.ID)
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
