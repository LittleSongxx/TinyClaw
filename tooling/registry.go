package tooling

import (
	"sort"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
)

type ToolSpec struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Policy      ToolPolicy `json:"policy"`
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

type Entry struct {
	Spec      ToolSpec
	AgentInfo *conf.AgentInfo
}

type Registry struct {
	entries map[string]*Entry
}

func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]*Entry),
	}
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
		specs = append(specs, entry.Spec)
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
	return entry, ok
}
