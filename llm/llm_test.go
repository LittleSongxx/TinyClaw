package llm

import (
	"context"
	"testing"
	"time"

	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/stretchr/testify/assert"
)

func TestSendMsg_WithMessageChan(t *testing.T) {
	msgChan := make(chan *param.MsgInfo, 1)
	l := &LLM{MessageChan: msgChan}
	msg := &param.MsgInfo{SendLen: 10}

	updated := l.SendMsg(msg, "hello")
	assert.Equal(t, "hello", updated.Content)
}

func TestSendMsg_WithHTTPMsgChan(t *testing.T) {
	httpChan := make(chan string, 1)
	l := &LLM{HTTPMsgChan: httpChan}

	l.SendMsg(&param.MsgInfo{}, "streamed text")
	select {
	case msg := <-httpChan:
		assert.Equal(t, "streamed text", msg)
	case <-time.After(time.Second):
		t.Error("Expected message on HTTPMsgChan")
	}
}

func TestOverLoop(t *testing.T) {
	l := &LLM{LoopNum: 14}
	assert.False(t, l.OverLoop())
	assert.Equal(t, 15, l.LoopNum)
	assert.True(t, l.OverLoop())
}

func TestNewLLM_DefaultsToClient(t *testing.T) {
	ctx := db.WithCtxUserInfo(context.Background(), &db.User{
		LLMConfig:    `{"type":"gemini"}`,
		LLMConfigRaw: &param.LLMConfig{TxtType: param.Gemini},
	})
	l := NewLLM(
		WithUserId("u1"),
		WithContent("ask"),
		WithModel("m1"),
		WithContext(ctx),
	)
	assert.NotNil(t, l)
	assert.Equal(t, "u1", l.UserId)
	assert.Equal(t, "ask", l.Content)
	assert.Equal(t, "m1", l.Model)
}
