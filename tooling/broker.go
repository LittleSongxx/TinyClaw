package tooling

import (
	"context"
	"errors"
	"sync"
)

const (
	CategoryHost      = "host"
	CategoryNode      = "node"
	CategoryMCP       = "mcp"
	CategoryBrowser   = "browser"
	CategoryKnowledge = "knowledge"
)

type ToolInvocation struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	NodeID    string                 `json:"node_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
}

type ToolProvider interface {
	Name() string
	ListTools(ctx context.Context) ([]ToolSpec, error)
	Supports(name string) bool
	ExecuteTool(ctx context.Context, call ToolInvocation) (*ToolResult, error)
}

type RuntimeVisibleProvider interface {
	ChatVisible() bool
}

var (
	ErrToolBrokerNil        = errors.New("tool broker is nil")
	ErrToolProviderNotFound = errors.New("tool provider not found")
)

type Broker struct {
	mu        sync.RWMutex
	providers []ToolProvider
}

func NewBroker(providers ...ToolProvider) *Broker {
	b := &Broker{}
	for _, provider := range providers {
		b.Register(provider)
	}
	return b
}

func (b *Broker) Register(provider ToolProvider) {
	if b == nil || provider == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.providers = append(b.providers, provider)
}

func (b *Broker) List(ctx context.Context) ([]ToolSpec, error) {
	if b == nil {
		return nil, nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	specs := make([]ToolSpec, 0)
	for _, provider := range b.providers {
		items, err := provider.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			specs = append(specs, normalizeToolSpec(item))
		}
	}
	return specs, nil
}

func (b *Broker) Execute(ctx context.Context, call ToolInvocation) (*ToolResult, error) {
	if b == nil {
		return nil, ErrToolBrokerNil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, provider := range b.providers {
		if provider.Supports(call.Name) {
			return provider.ExecuteTool(ctx, call)
		}
	}
	return nil, ErrToolProviderNotFound
}

func (b *Broker) ListExecutable(ctx context.Context) ([]ToolSpec, error) {
	if b == nil {
		return nil, nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	specs := make([]ToolSpec, 0)
	for _, provider := range b.providers {
		if visible, ok := provider.(RuntimeVisibleProvider); ok && !visible.ChatVisible() {
			continue
		}
		items, err := provider.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			specs = append(specs, normalizeToolSpec(item))
		}
	}
	return specs, nil
}

type RegistryProvider struct {
	registry *Registry
}

func NewRegistryProvider(registry *Registry) *RegistryProvider {
	return &RegistryProvider{registry: registry}
}

func (p *RegistryProvider) Name() string {
	return "legacy-task-tools"
}

func (p *RegistryProvider) ChatVisible() bool {
	return false
}

func (p *RegistryProvider) ListTools(ctx context.Context) ([]ToolSpec, error) {
	if p == nil || p.registry == nil {
		return nil, nil
	}
	return p.registry.List(), nil
}

func (p *RegistryProvider) Supports(name string) bool {
	if p == nil || p.registry == nil {
		return false
	}
	_, ok := p.registry.Get(name)
	return ok
}

func (p *RegistryProvider) ExecuteTool(ctx context.Context, call ToolInvocation) (*ToolResult, error) {
	return nil, errors.New("legacy task tools are exposed for planning only; execution remains in current llm runtime")
}
