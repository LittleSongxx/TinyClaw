package skill

import (
	"testing"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestParseSkill(t *testing.T) {
	body := `---
id: local_research
name: Local Research
description: Research a topic with local rules.
version: v1
modes: [task, mcp, skill]
triggers: [research, summarize]
allowed_tools: [fetch_page]
memory: both
max_steps: 4
timeout_sec: 90
priority: 10
---
## When to use
Use this skill for research requests.

## When not to use
Do not use this skill for browser automation.

## Instructions
Collect evidence before answering.

## Output contract
Return a concise evidence-backed answer.

## Failure handling
State what evidence is missing.
`

	item, err := Parse("/tmp/SKILL.md", body, "local")
	assert.NoError(t, err)
	if assert.NotNil(t, item) {
		assert.Equal(t, "local_research", item.Manifest.ID)
		assert.Equal(t, MemoryBoth, item.Manifest.Memory)
		assert.Equal(t, "Collect evidence before answering.", item.Sections.Instructions)
		assert.Equal(t, "Return a concise evidence-backed answer.", item.Sections.OutputContract)
	}
}

func TestParseSkillMissingSection(t *testing.T) {
	body := `---
id: broken_skill
name: Broken Skill
description: Missing sections.
modes: [task]
allowed_tools: [fetch_page]
memory: conversation
---
## When to use
Use it.

## Instructions
Do it.

## Output contract
Return something.

## Failure handling
Explain the failure.
`

	_, err := Parse("/tmp/SKILL.md", body, "local")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "When not to use")
}

func TestSelectSkillsPrefersTriggerMatches(t *testing.T) {
	catalog := &Catalog{
		skills: map[string]*Skill{},
		ordered: []*Skill{
			{
				Manifest: Manifest{
					ID:          "browser_operator",
					Name:        "Browser Operator",
					Description: "Browse websites",
					Modes:       []string{ModeMCP},
					Triggers:    []string{"browser", "website", "navigate"},
					Priority:    50,
				},
				Source: "builtin",
			},
			{
				Manifest: Manifest{
					ID:          "legacy_all_tools_proxy",
					Name:        "Legacy All Tools Proxy",
					Description: "Fallback",
					Modes:       []string{ModeMCP},
					Priority:    0,
				},
				Source: "legacy",
				Legacy: true,
			},
		},
	}
	for _, item := range catalog.ordered {
		catalog.skills[item.Manifest.ID] = item
	}

	selected := catalog.selectSkills(ModeMCP, "please navigate this website in the browser", "", 1)
	if assert.Len(t, selected, 2) {
		assert.Equal(t, "browser_operator", selected[0].Manifest.ID)
		assert.Equal(t, "legacy_all_tools_proxy", selected[1].Manifest.ID)
	}
}

func TestBuildRegistryUsesAllowedTools(t *testing.T) {
	catalog := &Catalog{
		skills: map[string]*Skill{
			"workspace_operator": {
				Manifest: Manifest{
					ID:           "workspace_operator",
					Name:         "Workspace Operator",
					Description:  "Read workspace files",
					Version:      "v1",
					Modes:        []string{ModeSkill},
					AllowedTools: []string{"read_file"},
					Memory:       MemoryConversation,
				},
				Sections: Sections{
					WhenToUse:       "Use for files.",
					WhenNotToUse:    "Do not use for websites.",
					Instructions:    "Inspect files carefully.",
					OutputContract:  "Return file findings.",
					FailureHandling: "Explain file errors.",
				},
				Path:   "builtin://workspace_operator",
				Source: "builtin",
			},
		},
		ordered: []*Skill{},
		toolsByName: map[string]mcp.Tool{
			"read_file": {
				Name:        "read_file",
				Description: "Read a file",
			},
		},
	}
	catalog.ordered = append(catalog.ordered, catalog.skills["workspace_operator"])

	registry := catalog.BuildRegistry(ModeSkill, "read file", "workspace_operator", 1)
	entry, ok := registry.Get("workspace_operator")
	if assert.True(t, ok) {
		assert.NotNil(t, entry.Skill)
		assert.Equal(t, []string{"read_file"}, entry.Skill.AllowedTools)
		assert.Len(t, entry.AgentInfo.OpenAITools, 1)
	}
}

func TestAddSkillExpandsAllowedServers(t *testing.T) {
	catalog := &Catalog{
		skills:  map[string]*Skill{},
		ordered: []*Skill{},
		toolsByName: map[string]mcp.Tool{
			"read_file":  {Name: "read_file", Description: "Read a file"},
			"write_file": {Name: "write_file", Description: "Write a file"},
		},
		serverTools: map[string][]string{
			"filesystem": {"read_file", "write_file"},
		},
	}

	item := &Skill{
		Manifest: Manifest{
			ID:             "workspace_operator",
			Name:           "Workspace Operator",
			Description:    "Work with files",
			Modes:          []string{ModeSkill},
			AllowedServers: []string{"filesystem"},
			Memory:         MemoryConversation,
		},
		Sections: Sections{
			WhenToUse:       "Use for files.",
			WhenNotToUse:    "Do not use for websites.",
			Instructions:    "Inspect files carefully.",
			OutputContract:  "Return file findings.",
			FailureHandling: "Explain file errors.",
		},
		Path:   "/tmp/skills/workspace_operator/SKILL.md",
		Source: "local",
	}

	err := catalog.addSkill(item)
	assert.NoError(t, err)
	assert.Equal(t, []string{"read_file", "write_file"}, catalog.skills["workspace_operator"].Manifest.AllowedTools)
}

func TestResolveSkillSupportsLegacyAlias(t *testing.T) {
	catalog := &Catalog{
		skills: map[string]*Skill{
			"legacy_amap_proxy": {
				Manifest: Manifest{
					ID:          "legacy_amap_proxy",
					Name:        "Legacy amap Proxy",
					Description: "AMap legacy proxy",
				},
				Path:   "legacy://amap",
				Source: "legacy",
				Legacy: true,
			},
		},
		ordered: []*Skill{},
	}
	catalog.ordered = append(catalog.ordered, catalog.skills["legacy_amap_proxy"])

	item, ok := catalog.ResolveSkill("amap")
	if assert.True(t, ok) && assert.NotNil(t, item) {
		assert.Equal(t, "legacy_amap_proxy", item.Manifest.ID)
	}
}

func TestAppendLegacySkillsRequiresFeatureFlag(t *testing.T) {
	previous := conf.FeatureConfInfo.LegacyMCPProxy
	defer func() {
		conf.FeatureConfInfo.LegacyMCPProxy = previous
	}()

	catalog := &Catalog{
		skills:  map[string]*Skill{},
		ordered: []*Skill{},
		toolsByName: map[string]mcp.Tool{
			"maps_search": {Name: "maps_search", Description: "Search map data"},
		},
		serverTools: map[string][]string{
			"amap": {"maps_search"},
		},
		serverDesc: map[string]string{
			"amap": "Map data",
		},
	}

	conf.FeatureConfInfo.LegacyMCPProxy = false
	catalog.appendLegacySkills()
	assert.Empty(t, catalog.ordered)

	conf.FeatureConfInfo.LegacyMCPProxy = true
	catalog.appendLegacySkills()
	assert.Contains(t, catalog.skills, "legacy_amap_proxy")
	assert.Contains(t, catalog.skills, "legacy_all_tools_proxy")
}

func TestFormatMCPList(t *testing.T) {
	catalog := &Catalog{
		serverDesc: map[string]string{
			"amap": "Map data",
		},
		serverTools: map[string][]string{
			"amap": {"maps_search", "maps_route"},
		},
	}

	content := FormatMCPList(catalog)
	assert.Contains(t, content, "Available MCP servers:")
	assert.Contains(t, content, "amap")
	assert.Contains(t, content, "maps_search, maps_route")
}
