package llm

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestOpenAISend(t *testing.T) {
	if os.Getenv("OPENAI_TOKEN") == "" {
		t.Skip("OPENAI_TOKEN is required for live OpenAI send test")
	}

	messageChan := make(chan *param.MsgInfo)

	go func() {
		for m := range messageChan {
			fmt.Println(m)
		}
	}()

	conf.BaseConfInfo.CustomUrl = os.Getenv("TEST_CUSTOM_URL")
	conf.BaseConfInfo.Type = param.OpenAi

	ctx := db.WithCtxUserInfo(context.Background(), &db.User{
		LLMConfig:    `{"type":"openai"}`,
		LLMConfigRaw: &param.LLMConfig{TxtType: param.OpenAi},
	})

	callLLM := NewLLM(WithChatId("1"), WithMsgId("2"), WithUserId("5"),
		WithMessageChan(messageChan), WithContent("hi"), WithContext(ctx))
	callLLM.LLMClient.GetModel(callLLM)
	callLLM.GetMessages("5", "hi")
	err := callLLM.LLMClient.Send(ctx, callLLM)
	assert.Equal(t, nil, err)

	conf.BaseConfInfo.CustomUrl = ""
}

func TestOpenAIReq_GetMessage(t *testing.T) {
	req := &OpenAIReq{}
	req.GetMessage("user", "hello")
	assert.Len(t, req.OpenAIMsgs, 1)
	assert.Equal(t, "hello", req.OpenAIMsgs[0].Content)

	req.GetMessage("assistant", "hi")
	assert.Len(t, req.OpenAIMsgs, 2)
	assert.Equal(t, "hi", req.OpenAIMsgs[1].Content)
}

func TestOpenAIReq_AppendMessages(t *testing.T) {
	req1 := &OpenAIReq{}
	req1.GetMessage("user", "message from req1")

	req2 := &OpenAIReq{}
	req2.AppendMessages(req1)

	assert.Len(t, req2.OpenAIMsgs, 1)
	assert.Equal(t, "message from req1", req2.OpenAIMsgs[0].Content)
}

func TestOpenAIReq_GetModel_Default(t *testing.T) {
	oldToken := conf.BaseConfInfo.OpenAIToken
	conf.BaseConfInfo.OpenAIToken = "test-openai-token"
	defer func() {
		conf.BaseConfInfo.OpenAIToken = oldToken
	}()

	req := &OpenAIReq{}
	ctx := db.WithCtxUserInfo(context.Background(), &db.User{
		LLMConfig:    `{"type":"openai"}`,
		LLMConfigRaw: &param.LLMConfig{TxtType: param.OpenAi},
	})
	llmObj := NewLLM(WithChatId("1"), WithMsgId("2"), WithUserId("4"),
		WithContent("hi"), WithContext(ctx))

	req.GetModel(llmObj)
	assert.Equal(t, openai.GPT3Dot5Turbo0125, llmObj.Model)
}

func TestRequestToolsCall_InvalidJSON(t *testing.T) {
	req := &OpenAIReq{
		ToolCall: []openai.ToolCall{},
	}

	streamChoice := openai.ChatCompletionStreamChoice{
		Delta: openai.ChatCompletionStreamChoiceDelta{
			ToolCalls: []openai.ToolCall{
				{
					ID:   "tool-id",
					Type: "function",
					Function: openai.FunctionCall{
						Name:      "mockTool",
						Arguments: "{invalid-json",
					},
				},
			},
		},
	}

	err := req.RequestToolsCall(context.Background(), streamChoice, nil)
	assert.Equal(t, ErrToolsJSON, err)
}

func TestRequestToolsCall_StoresArgumentsWhenNameAndArgumentsArriveTogether(t *testing.T) {
	req := &OpenAIReq{
		ToolCall: []openai.ToolCall{},
	}

	idx := 0
	streamChoice := openai.ChatCompletionStreamChoice{
		Delta: openai.ChatCompletionStreamChoiceDelta{
			ToolCalls: []openai.ToolCall{
				{
					Index: &idx,
					ID:    "tool-id",
					Type:  "function",
					Function: openai.FunctionCall{
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

func TestRequestToolsCall_KeepsToolCallsSeparatedByIndex(t *testing.T) {
	req := &OpenAIReq{
		ToolCall: []openai.ToolCall{},
	}

	firstIdx := 0
	secondIdx := 1

	err := req.RequestToolsCall(context.Background(), openai.ChatCompletionStreamChoice{
		Delta: openai.ChatCompletionStreamChoiceDelta{
			ToolCalls: []openai.ToolCall{
				{
					Index: &firstIdx,
					ID:    "tool-1",
					Type:  "function",
					Function: openai.FunctionCall{
						Name:      "toolOne",
						Arguments: "{\"first\":",
					},
				},
			},
		},
	}, nil)
	assert.Equal(t, ErrToolsJSON, err)

	err = req.RequestToolsCall(context.Background(), openai.ChatCompletionStreamChoice{
		Delta: openai.ChatCompletionStreamChoiceDelta{
			ToolCalls: []openai.ToolCall{
				{
					Index: &secondIdx,
					ID:    "tool-2",
					Type:  "function",
					Function: openai.FunctionCall{
						Name:      "toolTwo",
						Arguments: "{\"second\":",
					},
				},
			},
		},
	}, nil)
	assert.Equal(t, ErrToolsJSON, err)

	if assert.Len(t, req.ToolCall, 2) {
		assert.Equal(t, "toolOne", req.ToolCall[0].Function.Name)
		assert.Equal(t, "{\"first\":", req.ToolCall[0].Function.Arguments)
		assert.Equal(t, "toolTwo", req.ToolCall[1].Function.Name)
		assert.Equal(t, "{\"second\":", req.ToolCall[1].Function.Arguments)
	}
}
