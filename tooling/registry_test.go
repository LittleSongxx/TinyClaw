package tooling

import "testing"

func TestNewRegistryIncludesDirectLLMTool(t *testing.T) {
	registry := NewRegistry()

	entry, ok := registry.Get(DirectLLMToolName)
	if !ok || entry == nil {
		t.Fatalf("expected %s to be present in registry", DirectLLMToolName)
	}
	if entry.Spec.Description == "" {
		t.Fatalf("expected %s to include a description", DirectLLMToolName)
	}
	if entry.AgentInfo != nil {
		t.Fatalf("expected %s to be a direct tool without agent info", DirectLLMToolName)
	}
}
