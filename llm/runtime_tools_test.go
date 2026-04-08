package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/LittleSongxx/TinyClaw/conf"
	appi18n "github.com/LittleSongxx/TinyClaw/i18n"
	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/tooling"
)

type runtimeFakeTransport struct {
	result *node.NodeCommandResult
}

func (f *runtimeFakeTransport) Request(ctx context.Context, req node.NodeCommandRequest) (*node.NodeCommandResult, error) {
	if f.result == nil {
		return &node.NodeCommandResult{
			ID:         req.ID,
			NodeID:     req.NodeID,
			Capability: req.Capability,
			Success:    true,
			Output:     "ok",
		}, nil
	}
	result := *f.result
	result.ID = req.ID
	result.NodeID = req.NodeID
	result.Capability = req.Capability
	return &result, nil
}

func (f *runtimeFakeTransport) Close() error { return nil }

func TestEnsureRuntimeToolsAppendsNodeTools(t *testing.T) {
	manager := node.NewManager()
	err := manager.RegisterNode(context.Background(), node.NodeDescriptor{
		ID: "node-1",
		Capabilities: []node.NodeCapability{
			{Name: "browser.open"},
		},
	}, &runtimeFakeTransport{})
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	client := &LLM{
		Ctx:        context.Background(),
		ToolBroker: tooling.NewBroker(tooling.NewNodeProvider(manager)),
	}
	client.ensureRuntimeTools()

	found := false
	for _, tool := range client.OpenAITools {
		if tool.Function != nil && tool.Function.Name == "node_browser_open" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected node browser tool to be injected, got %+v", client.OpenAITools)
	}
	if len(client.RuntimeToolGuidance) == 0 {
		t.Fatalf("expected runtime guidance to be injected")
	}
}

func TestExecMcpReqUsesRuntimeToolBroker(t *testing.T) {
	manager := node.NewManager()
	err := manager.RegisterNode(context.Background(), node.NodeDescriptor{
		ID: "node-1",
		Capabilities: []node.NodeCapability{
			{Name: "browser.open"},
		},
	}, &runtimeFakeTransport{
		result: &node.NodeCommandResult{
			Success: true,
			Output:  "opened",
			Data: map[string]interface{}{
				"url": "https://example.com",
			},
		},
	})
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	client := &LLM{
		Ctx:        context.Background(),
		Cs:         &param.ContextState{SessionID: "session-1"},
		ToolBroker: tooling.NewBroker(tooling.NewNodeProvider(manager)),
	}

	output, err := client.ExecMcpReq(context.Background(), "node_browser_open", map[string]interface{}{
		"node_id": "node-1",
		"url":     "https://example.com",
	})
	if err != nil {
		t.Fatalf("execute runtime tool: %v", err)
	}
	if !strings.Contains(output, `"success":true`) || !strings.Contains(output, `https://example.com`) {
		t.Fatalf("unexpected runtime tool output: %s", output)
	}
}

func TestFinalizeToolResultHandlesPendingApproval(t *testing.T) {
	appi18n.InitI18n()
	oldLang := conf.BaseConfInfo.Lang
	conf.BaseConfInfo.Lang = "zh"
	defer func() {
		conf.BaseConfInfo.Lang = oldLang
	}()

	messageChan := make(chan *param.MsgInfo, 1)
	client := &LLM{
		MessageChan: messageChan,
		PerMsgLen:   4096,
	}

	output, err := client.finalizeToolResult("node_keyboard_type", map[string]interface{}{"text": "hello"}, `{"pending_approval":true,"approval_id":"approval-1","summary":"Type text on the paired PC"}`)
	if err != nil {
		t.Fatalf("finalize tool result: %v", err)
	}
	if !strings.Contains(output, "等待用户确认") {
		t.Fatalf("expected waiting placeholder, got %s", output)
	}

	select {
	case msg := <-messageChan:
		if !strings.Contains(msg.Content, "/approve approval-1") {
			t.Fatalf("expected approval instruction, got %s", msg.Content)
		}
	default:
		t.Fatalf("expected confirmation message to be sent")
	}
}

func TestSanitizeToolResponseForUserRemovesImagePlaceholders(t *testing.T) {
	raw := "]\n{这里只是一个示例 Base64 图像字符串，实际使用时应替换为 MCP 返回的图像数据}\n这是您当前 Windows 桌面的截图。请查看。"
	got := sanitizeToolResponseForUser(raw)
	if strings.Contains(got, "Base64 图像字符串") || strings.HasPrefix(got, "]") {
		t.Fatalf("expected placeholders to be removed, got %q", got)
	}
	if !strings.Contains(got, "这是您当前 Windows 桌面的截图") {
		t.Fatalf("expected human-readable text to remain, got %q", got)
	}
}

func TestBuildImageToolSummaryIncludesWindowTitle(t *testing.T) {
	got := buildImageToolSummary("node_screen_snapshot", map[string]interface{}{
		"scope":  "active_window",
		"width":  1280,
		"height": 720,
		"window": map[string]interface{}{
			"title": "记事本",
		},
	})
	if !strings.Contains(got, "active_window") || !strings.Contains(got, "记事本") || !strings.Contains(got, "1280x720") {
		t.Fatalf("expected active window summary with title and size, got %q", got)
	}
}
