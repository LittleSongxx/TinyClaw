package session

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFileStoreOpenAppendAndRecent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	store := NewFileStore(root)
	ctx := context.Background()

	env, err := store.Open(ctx, SessionKey{
		Channel:   "web",
		AccountID: "default",
		PeerID:    "u-1",
		Kind:      "dm",
	}, map[string]string{"source": "test"})
	if err != nil {
		t.Fatalf("open session: %v", err)
	}

	if _, err := store.Append(ctx, env.SessionID, Message{Role: RoleUser, Content: "hello"}); err != nil {
		t.Fatalf("append user: %v", err)
	}
	if _, err := store.Append(ctx, env.SessionID, Message{Role: RoleAssistant, Content: "world"}); err != nil {
		t.Fatalf("append assistant: %v", err)
	}

	recent, err := store.Recent(ctx, env.SessionID, 10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(recent))
	}
	if recent[0].Content != "hello" || recent[1].Content != "world" {
		t.Fatalf("unexpected transcript: %+v", recent)
	}
}
