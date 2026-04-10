package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ManifestFileName = "tinyclaw.plugin.json"

type Manifest struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Version          string                 `json:"version"`
	EnabledByDefault bool                   `json:"enabledByDefault"`
	Capabilities     []CapabilityManifest   `json:"capabilities"`
	ConfigSchema     map[string]interface{} `json:"configSchema,omitempty"`
	UIHints          map[string]interface{} `json:"uiHints,omitempty"`
	Skills           []string               `json:"skills,omitempty"`
	ToolPacks        []string               `json:"toolPacks,omitempty"`
	NodeDrivers      []string               `json:"nodeDrivers,omitempty"`
	Channels         []string               `json:"channels,omitempty"`
	RequiredFeatures []string               `json:"requiredFeatures,omitempty"`
}

type CapabilityManifest struct {
	Name        string                 `json:"name"`
	Category    string                 `json:"category,omitempty"`
	Risk        string                 `json:"risk,omitempty"`
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

func LoadManifest(path string) (*Manifest, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	if err := ValidateManifest(&manifest); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &manifest, nil
}

func DiscoverManifests(root string) ([]Manifest, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, nil
	}
	items := make([]Manifest, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() != ManifestFileName {
			return nil
		}
		manifest, err := LoadManifest(path)
		if err != nil {
			return err
		}
		items = append(items, *manifest)
		return nil
	})
	return items, err
}

func ValidateManifest(manifest *Manifest) error {
	if manifest == nil {
		return errors.New("manifest is nil")
	}
	manifest.ID = strings.TrimSpace(manifest.ID)
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Version = strings.TrimSpace(manifest.Version)
	if manifest.ID == "" {
		return errors.New("id is required")
	}
	if manifest.Name == "" {
		return errors.New("name is required")
	}
	if manifest.Version == "" {
		return errors.New("version is required")
	}
	for index := range manifest.Capabilities {
		manifest.Capabilities[index].Name = strings.TrimSpace(manifest.Capabilities[index].Name)
		if manifest.Capabilities[index].Name == "" {
			return fmt.Errorf("capabilities[%d].name is required", index)
		}
	}
	return nil
}
