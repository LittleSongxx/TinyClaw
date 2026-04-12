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

func TestVolSend(t *testing.T) {
	if os.Getenv("VOLC_AK") == "" || os.Getenv("VOLC_SK") == "" {
		t.Skip("VOLC_AK and VOLC_SK are required for live Volcengine send test")
	}

	messageChan := make(chan *param.MsgInfo)

	go func() {
		for m := range messageChan {
			fmt.Println(m)
		}
	}()

	conf.BaseConfInfo.Type = param.Vol

	ctx := db.WithCtxUserInfo(context.Background(), &db.User{
		LLMConfig:    `{"type":"vol"}`,
		LLMConfigRaw: &param.LLMConfig{TxtType: param.Vol},
	})

	callLLM := NewLLM(WithChatId("1"), WithMsgId("2"), WithUserId("7"),
		WithMessageChan(messageChan), WithContent("hi"), WithContext(ctx))
	callLLM.LLMClient.GetModel(callLLM)
	callLLM.GetMessages("7", "hi")
	err := callLLM.LLMClient.Send(ctx, callLLM)
	assert.Equal(t, nil, err)
}
