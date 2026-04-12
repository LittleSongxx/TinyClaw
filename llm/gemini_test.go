package llm

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/stretchr/testify/assert"
)

func TestGeminiSend(t *testing.T) {
	if os.Getenv("GEMINI_TOKEN") == "" {
		t.Skip("GEMINI_TOKEN is required for live Gemini send test")
	}

	conf.InitConf()
	messageChan := make(chan *param.MsgInfo)

	go func() {
		for m := range messageChan {
			fmt.Println(m)
		}
	}()

	conf.BaseConfInfo.Type = param.Gemini

	ctx := db.WithCtxUserInfo(context.Background(), &db.User{
		LLMConfig:    `{"type":"gemini"}`,
		LLMConfigRaw: &param.LLMConfig{TxtType: param.Gemini},
	})
	callLLM := NewLLM(WithChatId("1"), WithMsgId("2"), WithUserId("4"),
		WithMessageChan(messageChan), WithContent("hi"), WithContext(ctx))
	callLLM.LLMClient.GetModel(callLLM)
	callLLM.GetMessages("4", "hi")
	err := callLLM.LLMClient.Send(ctx, callLLM)
	assert.Equal(t, nil, err)

}

func TestGenerateGeminiText_EmptyAudio(t *testing.T) {
	ctx := db.WithCtxUserInfo(context.Background(), &db.User{
		LLMConfig:    `{"type":"gemini"}`,
		LLMConfigRaw: &param.LLMConfig{TxtType: param.Gemini},
	})
	text, _, err := GenerateGeminiText(ctx, []byte{})
	assert.Error(t, err)
	assert.Empty(t, text)
}

func TestGenerateGeminiImage_EmptyPrompt(t *testing.T) {
	ctx := db.WithCtxUserInfo(context.Background(), &db.User{
		LLMConfig:    `{"type":"gemini"}`,
		LLMConfigRaw: &param.LLMConfig{TxtType: param.Gemini},
	})
	image, _, err := GenerateGeminiImg(ctx, "", nil)
	assert.Error(t, err)
	assert.Nil(t, image)
}

func TestGenerateGeminiVideo_InvalidPrompt(t *testing.T) {
	ctx := db.WithCtxUserInfo(context.Background(), &db.User{
		LLMConfig:    `{"type":"gemini"}`,
		LLMConfigRaw: &param.LLMConfig{TxtType: param.Gemini},
	})
	video, _, err := GenerateGeminiVideo(ctx, "", nil)
	assert.Error(t, err)
	assert.Nil(t, video)
}
