package tooling

import (
	"sort"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
)

type ToolSpec struct {
	Name            string     `json:"name"`
	Category        string     `json:"category,omitempty"`
	Source          string     `json:"source,omitempty"`
	Visibility      string     `json:"visibility,omitempty"`
	ApprovalPolicy  string     `json:"approval_policy,omitempty"`
	Tags            []string   `json:"tags,omitempty"`
	DisabledReason  string     `json:"disabled_reason,omitempty"`
	Description     string     `json:"description"`
	InputSchema     any        `json:"input_schema,omitempty"`
	Version         string     `json:"version,omitempty"`
	Path            string     `json:"path,omitempty"`
	Memory          string     `json:"memory,omitempty"`
	WhenToUse       string     `json:"when_to_use,omitempty"`
	WhenNotToUse    string     `json:"when_not_to_use,omitempty"`
	Instructions    string     `json:"instructions,omitempty"`
	OutputContract  string     `json:"output_contract,omitempty"`
	FailureHandling string     `json:"failure_handling,omitempty"`
	AllowedTools    []string   `json:"allowed_tools,omitempty"`
	Triggers        []string   `json:"triggers,omitempty"`
	Legacy          bool       `json:"legacy"`
	Policy          ToolPolicy `json:"policy"`
}

type ToolPolicy struct {
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
	Disabled   bool          `json:"disabled"`
}

type Observation struct {
	Function  string                 `json:"function"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Output    string                 `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	CreatedAt int64                  `json:"created_at"`
}

type ToolResult struct {
	Name         string        `json:"name"`
	Output       string        `json:"output"`
	Error        string        `json:"error,omitempty"`
	Observations []Observation `json:"observations,omitempty"`
	StartedAt    int64         `json:"started_at"`
	CompletedAt  int64         `json:"completed_at"`
}

type SkillRuntime struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Path            string   `json:"path"`
	Mode            string   `json:"mode"`
	Description     string   `json:"description"`
	Memory          string   `json:"memory"`
	WhenToUse       string   `json:"when_to_use"`
	WhenNotToUse    string   `json:"when_not_to_use"`
	Instructions    string   `json:"instructions"`
	OutputContract  string   `json:"output_contract"`
	FailureHandling string   `json:"failure_handling"`
	AllowedTools    []string `json:"allowed_tools,omitempty"`
	Triggers        []string `json:"triggers,omitempty"`
	Legacy          bool     `json:"legacy"`
}

type Entry struct {
	Spec      ToolSpec
	AgentInfo *conf.AgentInfo
	Skill     *SkillRuntime
}

type Registry struct {
	entries map[string]*Entry
}

func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]*Entry),
	}
}

func (r *Registry) Put(entry *Entry) {
	if r == nil || entry == nil {
		return
	}

	key := entry.Spec.Name
	if key == "" {
		return
	}

	r.entries[key] = entry
}

func NewRegistryFromTaskTools() *Registry {
	registry := NewRegistry()

	conf.TaskTools.Range(func(name, value any) bool {
		key, ok := name.(string)
		if !ok {
			return true
		}

		agentInfo, ok := value.(*conf.AgentInfo)
		if !ok || agentInfo == nil {
			return true
		}

		registry.entries[key] = &Entry{
			Spec: ToolSpec{
				Name:        key,
				Description: agentInfo.Description,
				Policy: ToolPolicy{
					Timeout:    2 * time.Minute,
					MaxRetries: conf.BaseConfInfo.LLMRetryTimes,
				},
			},
			AgentInfo: agentInfo,
		}
		return true
	})

	return registry
}

func (r *Registry) List() []ToolSpec {
	if r == nil {
		return nil
	}

	specs := make([]ToolSpec, 0, len(r.entries))
	for _, entry := range r.entries {
		specs = append(specs, normalizeToolSpec(entry.Spec))
	}

	sort.Slice(specs, func(i, j int) bool {
		return specs[i].Name < specs[j].Name
	})

	return specs
}

func (r *Registry) Get(name string) (*Entry, bool) {
	if r == nil {
		return nil, false
	}

	entry, ok := r.entries[name]
	if ok && entry != nil {
		copyEntry := *entry
		copyEntry.Spec = normalizeToolSpec(copyEntry.Spec)
		return &copyEntry, true
	}
	return entry, ok
}

func normalizeToolSpec(spec ToolSpec) ToolSpec {
	if spec.Source == "" {
		switch spec.Category {
		case CategoryNode:
			spec.Source = "node"
		case CategoryMCP, CategoryBrowser:
			spec.Source = "mcp"
		case CategoryKnowledge:
			spec.Source = "knowledge"
		default:
			spec.Source = "system"
		}
	}

	if spec.Visibility == "" {
		switch spec.Category {
		case CategoryNode, CategoryKnowledge, CategoryMCP, CategoryBrowser:
			spec.Visibility = "runtime"
		default:
			spec.Visibility = "planner"
		}
	}

	if spec.ApprovalPolicy == "" {
		if spec.Category == CategoryNode {
			spec.ApprovalPolicy = "user_confirmation"
		} else {
			spec.ApprovalPolicy = "implicit"
		}
	}

	if spec.Policy.Disabled && spec.DisabledReason == "" {
		spec.DisabledReason = "disabled by policy"
	}

	if len(spec.Tags) == 0 {
		switch spec.Source {
		case "node":
			spec.Tags = []string{"runtime", "desktop"}
		case "knowledge":
			spec.Tags = []string{"runtime", "recall"}
		case "mcp":
			spec.Tags = []string{"runtime", "mcp"}
		}
	}

	return spec
}
