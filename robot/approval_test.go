package robot

import (
	"strings"
	"testing"

	"github.com/LittleSongxx/TinyClaw/node"
)

func TestParseInlineApprovalDecision(t *testing.T) {
	cases := []struct {
		input    string
		approved bool
		ok       bool
	}{
		{input: "确认", approved: true, ok: true},
		{input: "同意", approved: true, ok: true},
		{input: "取消", approved: false, ok: true},
		{input: "拒绝", approved: false, ok: true},
		{input: "继续写代码", approved: false, ok: false},
	}

	for _, tc := range cases {
		approved, ok := parseInlineApprovalDecision(tc.input)
		if approved != tc.approved || ok != tc.ok {
			t.Fatalf("parseInlineApprovalDecision(%q) = (%v, %v), want (%v, %v)", tc.input, approved, ok, tc.approved, tc.ok)
		}
	}
}

func TestSanitizeNodeDiagnosticTextRemovesPowerShellNoise(t *testing.T) {
	raw := "exit status 1: ui element not found\nAt C:\\Temp\\tinyclaw.ps1:371 char:31\nCategoryInfo : RuntimeException\nFullyQualifiedErrorId : ui element not found"
	got := sanitizeNodeDiagnosticText(raw)
	if got != "ui element not found" {
		t.Fatalf("unexpected sanitized text: %q", got)
	}
}

func TestFriendlyNodeErrorMapsKnownUIFailure(t *testing.T) {
	got := friendlyNodeError("exit status 1: ui element not found")
	if got == "" || got == "ui element not found" {
		t.Fatalf("expected friendly mapped error, got %q", got)
	}
}

func TestBuildLarkApprovalCardShowsSessionGrantOptionWhenAllowed(t *testing.T) {
	card := buildLarkApprovalCard("approval-1", "session-1", "Focus Notepad", []string{
		string(node.ApprovalModeAllowOnce),
		string(node.ApprovalModeAllowSession),
	}, true, "", "")
	content, err := card.JSON()
	if err != nil {
		t.Fatalf("serialize card: %v", err)
	}
	if !strings.Contains(content, "本次允许") || !strings.Contains(content, "本会话允许") || !strings.Contains(content, "拒绝") {
		t.Fatalf("expected full approval actions in card, got %s", content)
	}
}

func TestBuildLarkApprovalCardOmitsSessionGrantOptionWhenUnavailable(t *testing.T) {
	card := buildLarkApprovalCard("approval-1", "session-1", "Type text", []string{
		string(node.ApprovalModeAllowOnce),
	}, true, "", "")
	content, err := card.JSON()
	if err != nil {
		t.Fatalf("serialize card: %v", err)
	}
	if strings.Contains(content, "本会话允许") {
		t.Fatalf("expected session grant action to be omitted, got %s", content)
	}
}
