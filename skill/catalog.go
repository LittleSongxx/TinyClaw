package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/tooling"
	"github.com/LittleSongxx/mcp-client-go/clients"
	mcpParam "github.com/LittleSongxx/mcp-client-go/clients/param"
	mcpUtils "github.com/LittleSongxx/mcp-client-go/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"
)

const (
	ModeChat     = "chat"
	ModeTask     = "task"
	ModeMCP      = "mcp"
	ModeSkill    = "skill"
	ModeWorkflow = "workflow"

	MemoryNone         = "none"
	MemoryConversation = "conversation"
	MemoryLongTerm     = "long_term"
	MemoryBoth         = "both"
)

var (
	frontmatterPattern = regexp.MustCompile(`(?s)\A---\s*\n(.*?)\n---\s*\n?(.*)\z`)
	sectionHeadingExpr = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)
	identifierExpr     = regexp.MustCompile(`[^a-z0-9_]+`)
	validModes         = map[string]bool{
		ModeChat:     true,
		ModeTask:     true,
		ModeMCP:      true,
		ModeSkill:    true,
		ModeWorkflow: true,
	}
	validMemoryModes = map[string]bool{
		MemoryNone:         true,
		MemoryConversation: true,
		MemoryLongTerm:     true,
		MemoryBoth:         true,
	}
)

type Manifest struct {
	ID             string   `yaml:"id" json:"id"`
	Name           string   `yaml:"name" json:"name"`
	Version        string   `yaml:"version" json:"version"`
	Description    string   `yaml:"description" json:"description"`
	Modes          []string `yaml:"modes" json:"modes"`
	Triggers       []string `yaml:"triggers" json:"triggers,omitempty"`
	AllowedServers []string `yaml:"allowed_servers" json:"allowed_servers,omitempty"`
	AllowedTools   []string `yaml:"allowed_tools" json:"allowed_tools,omitempty"`
	Memory         string   `yaml:"memory" json:"memory"`
	MaxSteps       int      `yaml:"max_steps" json:"max_steps,omitempty"`
	TimeoutSec     int      `yaml:"timeout_sec" json:"timeout_sec,omitempty"`
	Priority       int      `yaml:"priority" json:"priority,omitempty"`
}

type Sections struct {
	WhenToUse       string `json:"when_to_use"`
	WhenNotToUse    string `json:"when_not_to_use"`
	Instructions    string `json:"instructions"`
	OutputContract  string `json:"output_contract"`
	FailureHandling string `json:"failure_handling"`
}

type Skill struct {
	Manifest Manifest `json:"manifest"`
	Sections Sections `json:"sections"`
	Body     string   `json:"body"`
	Path     string   `json:"path"`
	Source   string   `json:"source"`
	Legacy   bool     `json:"legacy"`
}

type Catalog struct {
	skills      map[string]*Skill
	ordered     []*Skill
	toolsByName map[string]mcp.Tool
	serverTools map[string][]string
	serverDesc  map[string]string
	Warnings    []string `json:"warnings,omitempty"`
}

type MCPServerInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ToolCount   int      `json:"tool_count"`
	Tools       []string `json:"tools,omitempty"`
}

type LoadOptions struct {
	SkillRoots  []string
	MCPConfPath string
}

func DefaultRoots() []string {
	return []string{conf.GetAbsPath("skills")}
}

func LoadCatalog(opts LoadOptions) (*Catalog, error) {
	catalog := &Catalog{
		skills:      make(map[string]*Skill),
		ordered:     make([]*Skill, 0),
		toolsByName: make(map[string]mcp.Tool),
		serverTools: make(map[string][]string),
		serverDesc:  make(map[string]string),
		Warnings:    make([]string, 0),
	}

	if err := catalog.loadTools(opts.MCPConfPath); err != nil {
		catalog.Warnings = append(catalog.Warnings, err.Error())
	}

	roots := opts.SkillRoots
	if len(roots) == 0 {
		roots = DefaultRoots()
	}
	for _, root := range roots {
		if err := catalog.loadRoot(root); err != nil {
			catalog.Warnings = append(catalog.Warnings, err.Error())
		}
	}

	catalog.appendBuiltinSkills()
	catalog.appendLegacySkills()
	catalog.sortSkills()
	return catalog, nil
}

func (c *Catalog) List() []*Skill {
	if c == nil {
		return nil
	}

	res := make([]*Skill, 0, len(c.ordered))
	for _, skill := range c.ordered {
		res = append(res, cloneSkill(skill))
	}
	return res
}

func (c *Catalog) Get(id string) (*Skill, bool) {
	if c == nil {
		return nil, false
	}
	skill, ok := c.skills[id]
	if !ok {
		return nil, false
	}
	return cloneSkill(skill), true
}

func (c *Catalog) ResolveSkill(ref string) (*Skill, bool) {
	if c == nil {
		return nil, false
	}

	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, false
	}

	if skill, ok := c.skills[ref]; ok {
		return cloneSkill(skill), true
	}

	normalized := normalizeIdentifier(ref)
	var matched *Skill
	bestScore := -1
	for _, item := range c.ordered {
		if item == nil {
			continue
		}

		score := resolveSkillMatchScore(item, normalized)
		if score <= bestScore {
			continue
		}
		bestScore = score
		matched = item
	}

	if matched == nil {
		return nil, false
	}
	return cloneSkill(matched), true
}

func (c *Catalog) MCPServers() []MCPServerInfo {
	if c == nil {
		return nil
	}

	names := make([]string, 0, len(c.serverTools))
	for name := range c.serverTools {
		names = append(names, name)
	}
	sort.Strings(names)

	servers := make([]MCPServerInfo, 0, len(names))
	for _, name := range names {
		tools := append([]string(nil), c.serverTools[name]...)
		servers = append(servers, MCPServerInfo{
			Name:        name,
			Description: strings.TrimSpace(c.serverDesc[name]),
			ToolCount:   len(tools),
			Tools:       tools,
		})
	}
	return servers
}

func (c *Catalog) BuildRegistry(mode, input, explicitSkillID string, maxCandidates int) *tooling.Registry {
	registry := tooling.NewRegistry()
	if c == nil {
		return registry
	}

	selected := c.selectSkills(mode, input, explicitSkillID, maxCandidates)
	for _, skill := range selected {
		entry, err := c.entryFromSkill(skill, mode)
		if err != nil {
			c.Warnings = append(c.Warnings, err.Error())
			continue
		}
		registry.Put(entry)
	}

	return registry
}

func BuildPrompt(entry *tooling.Entry, userTask string) string {
	return BuildPromptWithMemory(entry, userTask, MemoryContext{})
}

func (c *Catalog) loadTools(confPath string) error {
	if strings.TrimSpace(confPath) == "" {
		return nil
	}

	body, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("read mcp conf fail: %w", err)
	}

	config := new(mcpParam.McpClientGoConfig)
	if err = json.Unmarshal(body, config); err != nil {
		return fmt.Errorf("parse mcp conf fail: %w", err)
	}

	serverNames := make([]string, 0, len(config.McpServers))
	for name, serverConf := range config.McpServers {
		serverNames = append(serverNames, name)
		if serverConf != nil {
			c.serverDesc[name] = strings.TrimSpace(serverConf.Description)
		}
	}
	sort.Strings(serverNames)

	for _, name := range serverNames {
		mc, getErr := clients.GetMCPClient(name)
		if getErr != nil || mc == nil {
			c.Warnings = append(c.Warnings, fmt.Sprintf("mcp client %s unavailable: %v", name, getErr))
			continue
		}

		toolNames := make([]string, 0, len(mc.Tools))
		for _, tool := range mc.Tools {
			c.toolsByName[tool.Name] = tool
			toolNames = append(toolNames, tool.Name)
		}
		sort.Strings(toolNames)
		c.serverTools[name] = toolNames
	}

	return nil
}

func (c *Catalog) loadRoot(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat skill root %s fail: %w", root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("skill root %s is not a directory", root)
	}

	return filepath.Walk(root, func(path string, entry os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry == nil || entry.IsDir() {
			return nil
		}
		if filepath.Base(path) != "SKILL.md" {
			return nil
		}

		skill, parseErr := ParseFile(path)
		if parseErr != nil {
			c.Warnings = append(c.Warnings, parseErr.Error())
			return nil
		}
		if err := c.addSkill(skill); err != nil {
			c.Warnings = append(c.Warnings, err.Error())
		}
		return nil
	})
}

func ParseFile(path string) (*Skill, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill %s fail: %w", path, err)
	}
	return Parse(path, string(body), "local")
}

func Parse(path, body, source string) (*Skill, error) {
	match := frontmatterPattern.FindStringSubmatch(body)
	if len(match) != 3 {
		return nil, fmt.Errorf("skill %s missing yaml frontmatter", path)
	}

	manifest := Manifest{}
	if err := yaml.Unmarshal([]byte(match[1]), &manifest); err != nil {
		return nil, fmt.Errorf("parse skill frontmatter %s fail: %w", path, err)
	}
	normalizeManifest(&manifest)

	sections := parseSections(match[2])
	if err := validateSkill(path, manifest, sections); err != nil {
		return nil, err
	}

	return &Skill{
		Manifest: manifest,
		Sections: sections,
		Body:     strings.TrimSpace(match[2]),
		Path:     path,
		Source:   source,
		Legacy:   source == "legacy",
	}, nil
}

func normalizeManifest(manifest *Manifest) {
	if manifest == nil {
		return
	}
	manifest.ID = strings.TrimSpace(manifest.ID)
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Description = strings.TrimSpace(manifest.Description)
	manifest.Version = strings.TrimSpace(manifest.Version)
	if manifest.Version == "" {
		manifest.Version = "v1"
	}
	if manifest.Memory == "" {
		manifest.Memory = MemoryConversation
	}
	manifest.Memory = strings.ToLower(strings.TrimSpace(manifest.Memory))

	modes := make([]string, 0, len(manifest.Modes))
	seenModes := make(map[string]bool)
	for _, mode := range manifest.Modes {
		mode = strings.ToLower(strings.TrimSpace(mode))
		if mode == "" || seenModes[mode] {
			continue
		}
		seenModes[mode] = true
		modes = append(modes, mode)
	}
	manifest.Modes = modes

	manifest.Triggers = dedupeStrings(manifest.Triggers)
	manifest.AllowedServers = dedupeStrings(manifest.AllowedServers)
	manifest.AllowedTools = dedupeStrings(manifest.AllowedTools)
}

func validateSkill(path string, manifest Manifest, sections Sections) error {
	if manifest.ID == "" {
		return fmt.Errorf("skill %s missing id", path)
	}
	if manifest.Name == "" {
		return fmt.Errorf("skill %s missing name", path)
	}
	if manifest.Description == "" {
		return fmt.Errorf("skill %s missing description", path)
	}
	if len(manifest.Modes) == 0 {
		return fmt.Errorf("skill %s missing modes", path)
	}
	for _, mode := range manifest.Modes {
		if !validModes[mode] {
			return fmt.Errorf("skill %s has invalid mode %s", path, mode)
		}
	}
	if !validMemoryModes[manifest.Memory] {
		return fmt.Errorf("skill %s has invalid memory mode %s", path, manifest.Memory)
	}
	if strings.TrimSpace(sections.WhenToUse) == "" {
		return fmt.Errorf("skill %s missing section When to use", path)
	}
	if strings.TrimSpace(sections.WhenNotToUse) == "" {
		return fmt.Errorf("skill %s missing section When not to use", path)
	}
	if strings.TrimSpace(sections.Instructions) == "" {
		return fmt.Errorf("skill %s missing section Instructions", path)
	}
	if strings.TrimSpace(sections.OutputContract) == "" {
		return fmt.Errorf("skill %s missing section Output contract", path)
	}
	if strings.TrimSpace(sections.FailureHandling) == "" {
		return fmt.Errorf("skill %s missing section Failure handling", path)
	}
	return nil
}

func parseSections(body string) Sections {
	body = strings.TrimSpace(body)
	if body == "" {
		return Sections{}
	}

	indexes := sectionHeadingExpr.FindAllStringSubmatchIndex(body, -1)
	if len(indexes) == 0 {
		return Sections{}
	}

	sections := make(map[string]string)
	for idx, match := range indexes {
		titleStart, titleEnd := match[2], match[3]
		contentStart := match[1]
		contentEnd := len(body)
		if idx+1 < len(indexes) {
			contentEnd = indexes[idx+1][0]
		}
		title := canonicalSectionTitle(body[titleStart:titleEnd])
		content := strings.TrimSpace(body[contentStart:contentEnd])
		sections[title] = content
	}

	return Sections{
		WhenToUse:       sections["when_to_use"],
		WhenNotToUse:    sections["when_not_to_use"],
		Instructions:    sections["instructions"],
		OutputContract:  sections["output_contract"],
		FailureHandling: sections["failure_handling"],
	}
}

func canonicalSectionTitle(title string) string {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "when to use":
		return "when_to_use"
	case "when not to use":
		return "when_not_to_use"
	case "instructions":
		return "instructions"
	case "output contract":
		return "output_contract"
	case "failure handling":
		return "failure_handling"
	default:
		return strings.ToLower(strings.TrimSpace(title))
	}
}

func (c *Catalog) addSkill(skill *Skill) error {
	if c == nil || skill == nil {
		return nil
	}
	if _, exists := c.skills[skill.Manifest.ID]; exists {
		return fmt.Errorf("duplicate skill id %s from %s", skill.Manifest.ID, skill.Path)
	}

	declaredTools := append([]string(nil), skill.Manifest.AllowedTools...)
	declaredTools = append(declaredTools, c.toolsForServers(skill.Manifest.AllowedServers...)...)

	validTools := make([]string, 0, len(declaredTools))
	for _, toolName := range dedupeStrings(declaredTools) {
		if _, ok := c.toolsByName[toolName]; ok {
			validTools = append(validTools, toolName)
			continue
		}
		c.Warnings = append(c.Warnings, fmt.Sprintf("skill %s references unknown tool %s", skill.Manifest.ID, toolName))
	}
	skill.Manifest.AllowedTools = validTools

	c.skills[skill.Manifest.ID] = skill
	c.ordered = append(c.ordered, skill)
	return nil
}

func (c *Catalog) appendBuiltinSkills() {
	seeds := []struct {
		ID          string
		Name        string
		Description string
		Modes       []string
		Triggers    []string
		Memory      string
		Priority    int
		MaxSteps    int
		TimeoutSec  int
		Servers     []string
		Sections    Sections
	}{
		{
			ID:          "general_research",
			Name:        "General Research",
			Description: "Researches a topic using web, academic, time, and memory tools before producing a grounded answer.",
			Modes:       []string{ModeTask, ModeMCP, ModeSkill},
			Triggers:    []string{"research", "search", "find", "compare", "summarize"},
			Memory:      MemoryBoth,
			Priority:    80,
			MaxSteps:    8,
			TimeoutSec:  180,
			Servers:     []string{"fetch", "bocha-search", "arxiv", "time", "memory"},
			Sections: Sections{
				WhenToUse:       "Use this skill for research, information gathering, comparisons, and evidence-backed summaries.",
				WhenNotToUse:    "Do not use this skill for browser automation, local workspace editing, or GitHub-only tasks.",
				Instructions:    "Gather the minimum useful evidence, prefer direct sources when available, and keep the answer grounded in the retrieved material.",
				OutputContract:  "Return a concise answer that separates findings from uncertainty and highlights key evidence.",
				FailureHandling: "If retrieval is incomplete, explain what was missing and provide the best partial answer.",
			},
		},
		{
			ID:          "browser_operator",
			Name:        "Browser Operator",
			Description: "Navigates websites and performs browser-based inspection or interaction tasks.",
			Modes:       []string{ModeTask, ModeMCP, ModeSkill},
			Triggers:    []string{"browser", "website", "page", "click", "navigate"},
			Memory:      MemoryConversation,
			Priority:    75,
			MaxSteps:    6,
			TimeoutSec:  180,
			Servers:     []string{"playwright", "fetch", "time"},
			Sections: Sections{
				WhenToUse:       "Use this skill for browsing, page inspection, structured web interaction, and collecting on-page evidence.",
				WhenNotToUse:    "Do not use this skill for filesystem operations, GitHub-specific analysis, or pure knowledge summarization without browsing.",
				Instructions:    "Prefer deterministic browser actions, verify page state before reporting, and summarize the observed result after each important interaction.",
				OutputContract:  "Return the requested page findings or interaction result with enough detail to verify what happened.",
				FailureHandling: "If the page cannot be accessed or interacted with, report the blocking condition clearly.",
			},
		},
		{
			ID:          "workspace_operator",
			Name:        "Workspace Operator",
			Description: "Inspects and manipulates files available through the filesystem MCP tools.",
			Modes:       []string{ModeTask, ModeMCP, ModeSkill},
			Triggers:    []string{"file", "directory", "workspace", "read", "write"},
			Memory:      MemoryConversation,
			Priority:    70,
			MaxSteps:    6,
			TimeoutSec:  180,
			Servers:     []string{"filesystem", "memory", "time"},
			Sections: Sections{
				WhenToUse:       "Use this skill for local file inspection, workspace analysis, and other filesystem-driven tasks.",
				WhenNotToUse:    "Do not use this skill for browser-only workflows or GitHub API tasks.",
				Instructions:    "Stay within the available workspace paths, inspect before changing assumptions, and summarize relevant file evidence.",
				OutputContract:  "Return the requested workspace result with referenced file paths or observed filesystem facts.",
				FailureHandling: "If a path is unavailable or a filesystem action fails, report the exact failing path or operation.",
			},
		},
		{
			ID:          "github_operator",
			Name:        "GitHub Operator",
			Description: "Handles GitHub repository, PR, issue, and commit inspection tasks.",
			Modes:       []string{ModeTask, ModeMCP, ModeSkill},
			Triggers:    []string{"github", "pull request", "pr", "issue", "commit"},
			Memory:      MemoryConversation,
			Priority:    85,
			MaxSteps:    6,
			TimeoutSec:  180,
			Servers:     []string{"github", "fetch", "time"},
			Sections: Sections{
				WhenToUse:       "Use this skill for GitHub-centric repository, issue, PR, and commit workflows.",
				WhenNotToUse:    "Do not use this skill for generic browsing, local workspace tasks, or unrelated research.",
				Instructions:    "Prefer GitHub tools first, keep the result repository-specific, and summarize the relevant state succinctly.",
				OutputContract:  "Return the requested GitHub findings with the key repository objects and their current status.",
				FailureHandling: "If GitHub access is unavailable, explain whether the issue is authentication, connectivity, or missing resources.",
			},
		},
	}

	for _, seed := range seeds {
		if _, exists := c.skills[seed.ID]; exists {
			continue
		}

		allowedTools := c.toolsForServers(seed.Servers...)
		if len(allowedTools) == 0 {
			continue
		}

		skill := &Skill{
			Manifest: Manifest{
				ID:           seed.ID,
				Name:         seed.Name,
				Version:      "v1",
				Description:  seed.Description,
				Modes:        seed.Modes,
				Triggers:     seed.Triggers,
				AllowedTools: allowedTools,
				Memory:       seed.Memory,
				MaxSteps:     seed.MaxSteps,
				TimeoutSec:   seed.TimeoutSec,
				Priority:     seed.Priority,
			},
			Sections: seed.Sections,
			Path:     "builtin://" + seed.ID,
			Source:   "builtin",
		}
		c.skills[skill.Manifest.ID] = skill
		c.ordered = append(c.ordered, skill)
	}
}

func (c *Catalog) appendLegacySkills() {
	if !conf.FeatureConfInfo.LegacyMCPProxyEnabled() {
		return
	}

	serverNames := make([]string, 0, len(c.serverTools))
	for serverName := range c.serverTools {
		serverNames = append(serverNames, serverName)
	}
	sort.Strings(serverNames)

	for _, serverName := range serverNames {
		toolNames := append([]string(nil), c.serverTools[serverName]...)
		if len(toolNames) == 0 {
			continue
		}

		skillID := "legacy_" + normalizeIdentifier(serverName) + "_proxy"
		if _, exists := c.skills[skillID]; exists {
			continue
		}

		description := strings.TrimSpace(c.serverDesc[serverName])
		if description == "" {
			description = fmt.Sprintf("Legacy proxy skill for MCP server %s.", serverName)
		}

		skill := &Skill{
			Manifest: Manifest{
				ID:           skillID,
				Name:         "Legacy " + serverName + " Proxy",
				Version:      "v1",
				Description:  description,
				Modes:        []string{ModeTask, ModeMCP, ModeSkill},
				AllowedTools: toolNames,
				Memory:       MemoryConversation,
				MaxSteps:     4,
				TimeoutSec:   120,
				Priority:     10,
			},
			Sections: Sections{
				WhenToUse:       "Use this legacy proxy when a task clearly maps to this MCP server and no stronger dedicated skill is a better fit.",
				WhenNotToUse:    "Do not use this proxy when a specialized non-legacy skill is available for the same job.",
				Instructions:    "Act as a compatibility proxy, use only the server tools, and keep the result aligned with the user request.",
				OutputContract:  "Return the practical result requested by the user without adding unrelated workflow.",
				FailureHandling: "If the underlying MCP server cannot complete the request, surface that failure directly.",
			},
			Path:   "legacy://" + serverName,
			Source: "legacy",
			Legacy: true,
		}
		c.skills[skill.Manifest.ID] = skill
		c.ordered = append(c.ordered, skill)
	}

	allTools := make([]string, 0, len(c.toolsByName))
	for name := range c.toolsByName {
		allTools = append(allTools, name)
	}
	sort.Strings(allTools)
	if len(allTools) == 0 {
		return
	}
	if _, exists := c.skills["legacy_all_tools_proxy"]; exists {
		return
	}

	c.skills["legacy_all_tools_proxy"] = &Skill{
		Manifest: Manifest{
			ID:           "legacy_all_tools_proxy",
			Name:         "Legacy All Tools Proxy",
			Version:      "v1",
			Description:  "Catch-all compatibility skill that exposes every currently registered MCP tool.",
			Modes:        []string{ModeTask, ModeMCP, ModeSkill},
			AllowedTools: allTools,
			Memory:       MemoryConversation,
			MaxSteps:     6,
			TimeoutSec:   180,
			Priority:     0,
		},
		Sections: Sections{
			WhenToUse:       "Use this skill only as a fallback when no more specific skill matches the task.",
			WhenNotToUse:    "Do not use this fallback when a dedicated or legacy single-server skill can solve the task more precisely.",
			Instructions:    "Stay focused on the user request and use the minimum tools required to finish it.",
			OutputContract:  "Return the final answer or result directly.",
			FailureHandling: "If the task still cannot be completed with the available tools, explain what is missing.",
		},
		Path:   "legacy://all_tools",
		Source: "legacy",
		Legacy: true,
	}
	c.ordered = append(c.ordered, c.skills["legacy_all_tools_proxy"])
}

func (c *Catalog) selectSkills(mode, input, explicitSkillID string, maxCandidates int) []*Skill {
	if c == nil {
		return nil
	}
	if explicitSkillID != "" {
		if skill, ok := c.skills[explicitSkillID]; ok {
			return []*Skill{skill}
		}
		return nil
	}

	type candidate struct {
		skill *Skill
		score int
	}

	normalized := strings.ToLower(strings.TrimSpace(input))
	candidates := make([]candidate, 0, len(c.ordered))
	var fallback *Skill

	for _, skill := range c.ordered {
		if skill == nil || !supportsMode(skill.Manifest.Modes, mode) {
			continue
		}
		if skill.Manifest.ID == "legacy_all_tools_proxy" {
			fallback = skill
		}

		score := skill.Manifest.Priority * 100
		if skill.Source == "local" {
			score += 30
		}
		if skill.Source == "builtin" {
			score += 20
		}
		if skill.Legacy {
			score -= 500
		}

		triggerHits := 0
		for _, trigger := range skill.Manifest.Triggers {
			if trigger != "" && strings.Contains(normalized, strings.ToLower(trigger)) {
				triggerHits++
			}
		}
		score += triggerHits * 25
		score += len(skill.Manifest.AllowedTools)

		candidates = append(candidates, candidate{
			skill: skill,
			score: score,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].skill.Manifest.ID < candidates[j].skill.Manifest.ID
		}
		return candidates[i].score > candidates[j].score
	})

	if maxCandidates <= 0 || maxCandidates > len(candidates) {
		maxCandidates = len(candidates)
	}

	res := make([]*Skill, 0, maxCandidates+1)
	seen := make(map[string]bool)
	for _, candidate := range candidates[:maxCandidates] {
		res = append(res, candidate.skill)
		seen[candidate.skill.Manifest.ID] = true
	}

	if fallback != nil && !seen[fallback.Manifest.ID] {
		res = append(res, fallback)
	}
	return res
}

func (c *Catalog) entryFromSkill(skill *Skill, mode string) (*tooling.Entry, error) {
	if skill == nil {
		return nil, fmt.Errorf("skill is nil")
	}
	tools := make([]mcp.Tool, 0, len(skill.Manifest.AllowedTools))
	for _, toolName := range skill.Manifest.AllowedTools {
		tool, ok := c.toolsByName[toolName]
		if !ok {
			continue
		}
		tools = append(tools, tool)
	}

	agentInfo := &conf.AgentInfo{
		Description:     skill.Manifest.Description,
		DeepseekTool:    mcpUtils.TransToolsToDPFunctionCall(tools),
		VolTool:         mcpUtils.TransToolsToVolFunctionCall(tools),
		OpenAITools:     mcpUtils.TransToolsToChatGPTFunctionCall(tools),
		GeminiTools:     mcpUtils.TransToolsToGeminiFunctionCall(tools),
		OpenRouterTools: mcpUtils.TransToolsToOpenRouterFunctionCall(tools),
	}

	timeout := 2 * time.Minute
	if skill.Manifest.TimeoutSec > 0 {
		timeout = time.Duration(skill.Manifest.TimeoutSec) * time.Second
	}

	return &tooling.Entry{
		Spec: tooling.ToolSpec{
			Name:            skill.Manifest.ID,
			Description:     skill.Manifest.Description,
			Version:         skill.Manifest.Version,
			Path:            skill.Path,
			Memory:          skill.Manifest.Memory,
			WhenToUse:       skill.Sections.WhenToUse,
			WhenNotToUse:    skill.Sections.WhenNotToUse,
			Instructions:    skill.Sections.Instructions,
			OutputContract:  skill.Sections.OutputContract,
			FailureHandling: skill.Sections.FailureHandling,
			AllowedTools:    append([]string(nil), skill.Manifest.AllowedTools...),
			Triggers:        append([]string(nil), skill.Manifest.Triggers...),
			Legacy:          skill.Legacy,
			Policy: tooling.ToolPolicy{
				Timeout:    timeout,
				MaxRetries: conf.BaseConfInfo.LLMRetryTimes,
			},
		},
		AgentInfo: agentInfo,
		Skill: &tooling.SkillRuntime{
			ID:              skill.Manifest.ID,
			Name:            skill.Manifest.Name,
			Version:         skill.Manifest.Version,
			Path:            skill.Path,
			Mode:            mode,
			Description:     skill.Manifest.Description,
			Memory:          skill.Manifest.Memory,
			WhenToUse:       skill.Sections.WhenToUse,
			WhenNotToUse:    skill.Sections.WhenNotToUse,
			Instructions:    skill.Sections.Instructions,
			OutputContract:  skill.Sections.OutputContract,
			FailureHandling: skill.Sections.FailureHandling,
			AllowedTools:    append([]string(nil), skill.Manifest.AllowedTools...),
			Triggers:        append([]string(nil), skill.Manifest.Triggers...),
			Legacy:          skill.Legacy,
		},
	}, nil
}

func (c *Catalog) toolsForServers(serverNames ...string) []string {
	res := make([]string, 0)
	seen := make(map[string]bool)
	for _, serverName := range serverNames {
		for _, toolName := range c.serverTools[serverName] {
			if seen[toolName] {
				continue
			}
			seen[toolName] = true
			res = append(res, toolName)
		}
	}
	sort.Strings(res)
	return res
}

func (c *Catalog) sortSkills() {
	sort.Slice(c.ordered, func(i, j int) bool {
		if c.ordered[i].Source == c.ordered[j].Source {
			return c.ordered[i].Manifest.ID < c.ordered[j].Manifest.ID
		}
		return c.ordered[i].Source < c.ordered[j].Source
	})
}

func supportsMode(modes []string, mode string) bool {
	if len(modes) == 0 {
		return false
	}
	for _, current := range modes {
		if current == mode {
			return true
		}
	}
	return false
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]bool)
	res := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		res = append(res, value)
	}
	sort.Strings(res)
	return res
}

func normalizeIdentifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = identifierExpr.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return "skill"
	}
	return value
}

func cloneSkill(skill *Skill) *Skill {
	if skill == nil {
		return nil
	}
	cloned := *skill
	cloned.Manifest.Modes = append([]string(nil), skill.Manifest.Modes...)
	cloned.Manifest.Triggers = append([]string(nil), skill.Manifest.Triggers...)
	cloned.Manifest.AllowedTools = append([]string(nil), skill.Manifest.AllowedTools...)
	return &cloned
}

func legacyServerName(skill *Skill) string {
	if skill == nil {
		return ""
	}

	path := strings.TrimSpace(skill.Path)
	if strings.HasPrefix(path, "legacy://") {
		return strings.TrimPrefix(path, "legacy://")
	}
	return ""
}

func skillAlias(skill *Skill) string {
	server := legacyServerName(skill)
	if server == "" || server == "all_tools" {
		return ""
	}
	return server
}

func resolveSkillMatchScore(skill *Skill, normalized string) int {
	if skill == nil || normalized == "" {
		return -1
	}

	best := -1
	if normalizeIdentifier(skill.Manifest.ID) == normalized {
		best = 400
	}
	if normalizeIdentifier(skill.Manifest.Name) == normalized && best < 300 {
		best = 300
	}
	if alias := skillAlias(skill); alias != "" && normalizeIdentifier(alias) == normalized && best < 350 {
		best = 350
	}
	return best
}
