package llm

import (
	"context"
	"testing"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/param"
	deepseek "github.com/cohesion-org/deepseek-go"
	"github.com/stretchr/testify/assert"
)

func TestOllamaReqGetMessage(t *testing.T) {
	req := &OllamaReq{}
	req.GetMessage("user", "hello")
	assert.Len(t, req.OllamaMsgs, 1)
	assert.Equal(t, "hello", req.OllamaMsgs[0].Content)

	req.GetMessage("assistant", "hi")
	assert.Len(t, req.OllamaMsgs, 2)
	assert.Equal(t, "hi", req.OllamaMsgs[1].Content)
}

func TestOllamaReqAppendMessages(t *testing.T) {
	req1 := &OllamaReq{}
	req1.GetMessage("user", "message from req1")

	req2 := &OllamaReq{}
	req2.AppendMessages(req1)

	assert.Len(t, req2.OllamaMsgs, 1)
	assert.Equal(t, "message from req1", req2.OllamaMsgs[0].Content)
}

func TestOllamaReqRequestToolsCallStoresArgumentsWhenNameAndArgumentsArriveTogether(t *testing.T) {
	req := &OllamaReq{ToolCall: []deepseek.ToolCall{}}

	streamChoice := deepseek.StreamChoices{
		Delta: deepseek.StreamDelta{
			ToolCalls: []deepseek.ToolCall{
				{
					ID:    "tool-id",
					Type:  "function",
					Index: 0,
					Function: deepseek.ToolCallFunction{
						Name:      "mockTool",
						Arguments: "{\"value\":",
					},
				},
			},
		},
	}

	err := req.RequestToolsCall(context.Background(), streamChoice, nil)
	assert.Equal(t, ErrToolsJSON, err)
	if assert.Len(t, req.ToolCall, 1) {
		assert.Equal(t, "mockTool", req.ToolCall[0].Function.Name)
		assert.Equal(t, "{\"value\":", req.ToolCall[0].Function.Arguments)
	}
}

func TestGetDeepseekClientDoesNotMutateConfiguredToken(t *testing.T) {
	oldToken := conf.BaseConfInfo.DeepseekToken
	oldType := conf.BaseConfInfo.Type
	conf.BaseConfInfo.DeepseekToken = "real-deepseek-token"
	conf.BaseConfInfo.Type = param.Ollama
	defer func() {
		conf.BaseConfInfo.DeepseekToken = oldToken
		conf.BaseConfInfo.Type = oldType
	}()

	ctx := db.WithCtxUserInfo(context.Background(), &db.User{
		LLMConfig:    `{"type":"ollama"}`,
		LLMConfigRaw: &param.LLMConfig{TxtType: param.Ollama},
	})

	client := GetDeepseekClient(ctx)
	if assert.NotNil(t, client) {
		assert.Equal(t, "http://localhost:11434/", client.BaseURL)
		assert.Equal(t, "api/chat", client.Path)
	}
	assert.Equal(t, "real-deepseek-token", conf.BaseConfInfo.DeepseekToken)
}
