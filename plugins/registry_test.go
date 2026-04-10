package plugins

import "testing"

func TestRegistryRejectsManifestWithoutStaticRegistrar(t *testing.T) {
	registry := NewDefaultRegistry()
	err := registry.LoadManifests([]Manifest{{
		ID:      "external-go-plugin",
		Name:    "External Go Plugin",
		Version: "1.0.0",
		Capabilities: []CapabilityManifest{
			{Name: "external.run"},
		},
	}})
	if err == nil {
		t.Fatal("expected manifest without compiled registrar to be rejected")
	}
}

func TestDefaultRegistryContainsBuiltInManifests(t *testing.T) {
	registry := NewDefaultRegistry()
	for _, id := range []string{"core-skills", "pc-node-tools", "browser-tools"} {
		if _, ok := registry.manifests[id]; !ok {
			t.Fatalf("expected built-in manifest %s", id)
		}
	}
}
