package robot

import "testing"

func TestCanonicalizeCommandNormalizesCaseAndAliases(t *testing.T) {
	cases := map[string]string{
		"/MCP":    "/mcp",
		"$SKILL":  "$skill",
		"/skil":   "/skill",
		"/skills": "/skill",
		"mcp":     "mcp",
	}

	for input, want := range cases {
		if got := canonicalizeCommand(input); got != want {
			t.Fatalf("canonicalizeCommand(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsExplicitCommand(t *testing.T) {
	if !isExplicitCommand("/amap") {
		t.Fatalf("expected slash command to be explicit")
	}
	if !isExplicitCommand("$skill") {
		t.Fatalf("expected dollar command to be explicit")
	}
	if isExplicitCommand("查询哈尔滨工业大学坐标") {
		t.Fatalf("did not expect plain text to be treated as command")
	}
}
