package skill

import (
	"context"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/recall"
	"github.com/LittleSongxx/TinyClaw/tooling"
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
		result.Conversation = recall.DefaultService().BuildConversationContext(userID, 4)
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
		longTerm, obs := recall.DefaultService().SearchLongTermMemory(ctx, userID, userTask, entry.Skill.ID)
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

	observation := recall.DefaultService().StoreLongTermMemory(ctx, userID, entry.Skill.ID, userTask, output)
	if observation.Function == "" {
		return nil
	}
	return []tooling.Observation{observation}
}

func BuildConversationContext(userID string, maxPairs int) string {
	return recall.DefaultService().BuildConversationContext(userID, maxPairs)
}

func searchLongTermMemory(ctx context.Context, userTask string, entry *tooling.Entry) (string, tooling.Observation) {
	if entry == nil || entry.Skill == nil {
		return "", tooling.Observation{}
	}
	return recall.DefaultService().SearchLongTermMemory(ctx, "", userTask, entry.Skill.ID)
}

func storeLongTermMemory(ctx context.Context, userID, userTask, output string, entry *tooling.Entry) tooling.Observation {
	if entry == nil || entry.Skill == nil {
		return tooling.Observation{}
	}
	return recall.DefaultService().StoreLongTermMemory(ctx, userID, entry.Skill.ID, userTask, output)
}
