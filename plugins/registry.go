package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/db"
)

type Registrar interface {
	ID() string
	Manifest() Manifest
}

type StaticRegistrar struct {
	PluginManifest Manifest
}

func (r StaticRegistrar) ID() string        { return r.PluginManifest.ID }
func (r StaticRegistrar) Manifest() Manifest { return r.PluginManifest }

type Status struct {
	Manifest
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason,omitempty"`
}

type Registry struct {
	mu         sync.RWMutex
	registrars map[string]Registrar
	manifests  map[string]Manifest
}

func NewRegistry(registrars ...Registrar) *Registry {
	registry := &Registry{
		registrars: make(map[string]Registrar),
		manifests:  make(map[string]Manifest),
	}
	for _, registrar := range registrars {
		_ = registry.Register(registrar)
	}
	return registry
}

func NewDefaultRegistry() *Registry {
	return NewRegistry(
		StaticRegistrar{PluginManifest: coreSkillsManifest()},
		StaticRegistrar{PluginManifest: pcNodeToolsManifest()},
		StaticRegistrar{PluginManifest: browserToolsManifest()},
	)
}

func (r *Registry) Register(registrar Registrar) error {
	if r == nil || registrar == nil {
		return nil
	}
	manifest := registrar.Manifest()
	if err := ValidateManifest(&manifest); err != nil {
		return err
	}
	if manifest.ID != registrar.ID() {
		return errors.New("manifest id does not match static registrar id")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registrars[manifest.ID] = registrar
	r.manifests[manifest.ID] = manifest
	return nil
}

func (r *Registry) LoadManifests(items []Manifest) error {
	for _, manifest := range items {
		if err := ValidateManifest(&manifest); err != nil {
			return err
		}
		r.mu.Lock()
		if _, ok := r.registrars[manifest.ID]; !ok {
			r.mu.Unlock()
			return errors.New("manifest has no compiled static registrar: " + manifest.ID)
		}
		r.manifests[manifest.ID] = manifest
		r.mu.Unlock()
	}
	return nil
}

func (r *Registry) List(ctx context.Context) ([]Status, error) {
	if r == nil {
		return nil, nil
	}
	workspaceID := authz.WorkspaceIDFromContext(ctx)
	states, err := db.ListPluginStates(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	r.mu.RLock()
	items := make([]Status, 0, len(r.manifests))
	for _, manifest := range r.manifests {
		state, ok := states[manifest.ID]
		isEnabled := state.Enabled
		if !ok {
			isEnabled = manifest.EnabledByDefault
		}
		items = append(items, Status{Manifest: manifest, Enabled: isEnabled})
	}
	r.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (r *Registry) Status(ctx context.Context, pluginID string) (*Status, error) {
	pluginID = strings.TrimSpace(pluginID)
	items, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ID == pluginID {
			return &item, nil
		}
	}
	return nil, errors.New("plugin not found")
}

func (r *Registry) SetEnabled(ctx context.Context, pluginID string, enabled bool, config string) error {
	principal, err := authz.RequirePrincipal(ctx)
	if err != nil {
		return err
	}
	if !principal.CanManageWorkspace() {
		return authz.ErrForbidden
	}
	r.mu.RLock()
	_, ok := r.manifests[pluginID]
	r.mu.RUnlock()
	if !ok {
		return errors.New("plugin not found or not registered")
	}
	if strings.TrimSpace(config) == "" {
		config = "{}"
	}
	var configMap map[string]interface{}
	_ = json.Unmarshal([]byte(config), &configMap)
	if err := db.UpsertPluginState(ctx, db.PluginState{
		WorkspaceID: principal.WorkspaceID,
		PluginID:   pluginID,
		Enabled:    enabled,
		Config:     configMap,
	}); err != nil {
		return err
	}
	action := "plugins.disable"
	if enabled {
		action = "plugins.enable"
	}
	_ = db.InsertAuditEvent(ctx, db.AuditEvent{
		WorkspaceID:  principal.WorkspaceID,
		ActorID:      principal.ActorID,
		Action:       action,
		ResourceType: "plugin",
		ResourceID:   pluginID,
		Success:      true,
	})
	return nil
}

func (r *Registry) Validate(ctx context.Context, manifest Manifest) error {
	if err := ValidateManifest(&manifest); err != nil {
		return err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.registrars[manifest.ID]; !ok {
		return errors.New("manifest has no compiled static registrar: " + manifest.ID)
	}
	return nil
}

func coreSkillsManifest() Manifest {
	return Manifest{
		ID:               "core-skills",
		Name:             "Core Skills",
		Version:          "1.0.0",
		EnabledByDefault: true,
		Capabilities: []CapabilityManifest{
			{Name: "skills.core", Category: "skill", Risk: "low"},
		},
		Skills: []string{"core"},
	}
}

func pcNodeToolsManifest() Manifest {
	return Manifest{
		ID:               "pc-node-tools",
		Name:             "PC Node Tools",
		Version:          "1.0.0",
		EnabledByDefault: false,
		Capabilities: []CapabilityManifest{
			{Name: "screen.snapshot", Category: "node", Risk: "low"},
			{Name: "system.exec", Category: "node", Risk: "high"},
			{Name: "fs.read", Category: "node", Risk: "low"},
			{Name: "fs.write", Category: "node", Risk: "high"},
		},
		NodeDrivers: []string{"windows", "wsl"},
	}
}

func browserToolsManifest() Manifest {
	return Manifest{
		ID:               "browser-tools",
		Name:             "Browser Tools",
		Version:          "1.0.0",
		EnabledByDefault: false,
		Capabilities: []CapabilityManifest{
			{Name: "browser.navigate", Category: "browser", Risk: "medium"},
			{Name: "browser.extract", Category: "browser", Risk: "low"},
		},
		ToolPacks: []string{"browser"},
	}
}
