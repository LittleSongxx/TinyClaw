package skill

import (
	"strings"
	"testing"

	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/tooling"
	"github.com/stretchr/testify/assert"
)

func TestBuildConversationContextUsesLatestPairs(t *testing.T) {
	userID := "skill-context-user"
	db.MsgRecord.Store(userID, &db.MsgRecordInfo{
		AQs: []*db.AQ{
			{Question: "q1", Answer: "a1"},
			{Question: "q2", Answer: "a2"},
			{Question: "q3", Answer: "a3"},
		},
	})
	t.Cleanup(func() {
		db.MsgRecord.Delete(userID)
	})

	ctx := BuildConversationContext(userID, 2)
	assert.NotContains(t, ctx, "q1")
	assert.Contains(t, ctx, "q2")
	assert.Contains(t, ctx, "a3")
}

func TestBuildPromptWithMemoryIncludesMemorySections(t *testing.T) {
	entry := &tooling.Entry{
		Skill: &tooling.SkillRuntime{
			ID:             "general_research",
			Name:           "General Research",
			Description:    "Research topics",
			Memory:         MemoryBoth,
			WhenToUse:      "Use for research.",
			WhenNotToUse:   "Do not use for browsing.",
			Instructions:   "Collect evidence.",
			OutputContract: "Return findings.",
			AllowedTools:   []string{"fetch_page"},
		},
	}

	prompt := BuildPromptWithMemory(entry, "Find the latest paper", MemoryContext{
		Conversation: "User: prior question",
		LongTerm:     "Past note",
	})

	assert.True(t, strings.Contains(prompt, "Conversation Memory"))
	assert.True(t, strings.Contains(prompt, "Long-Term Memory"))
	assert.True(t, strings.Contains(prompt, "Find the latest paper"))
}
