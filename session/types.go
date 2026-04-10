package session

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/LittleSongxx/TinyClaw/authz"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

type SessionKey struct {
	WorkspaceID string `json:"workspace_id"`
	Channel     string `json:"channel"`
	AccountID   string `json:"account_id"`
	PeerID      string `json:"peer_id"`
	GroupID     string `json:"group_id"`
	ThreadID    string `json:"thread_id"`
	Kind        string `json:"kind"`
	Nonce       string `json:"nonce,omitempty"`
}

func (k SessionKey) Scope() string {
	switch k.Kind {
	case "cron", "debug":
		return k.Kind
	}
	if k.GroupID != "" || k.ThreadID != "" {
		return "group"
	}
	return "dm"
}

func (k SessionKey) StableKey() string {
	workspaceID := authz.NormalizeWorkspaceID(k.WorkspaceID)
	parts := []string{
		workspaceID,
		strings.TrimSpace(k.Channel),
		strings.TrimSpace(k.AccountID),
		strings.TrimSpace(k.Scope()),
		strings.TrimSpace(k.PeerID),
		strings.TrimSpace(k.GroupID),
		strings.TrimSpace(k.ThreadID),
		strings.TrimSpace(k.Nonce),
	}
	return strings.Join(parts, "::")
}

func (k SessionKey) Hash() string {
	sum := sha1.Sum([]byte(k.StableKey()))
	return hex.EncodeToString(sum[:])
}

type Message struct {
	ID        string            `json:"id"`
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	CreatedAt int64             `json:"created_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type Envelope struct {
	WorkspaceID    string            `json:"workspace_id"`
	SessionID      string            `json:"session_id"`
	SessionKey     string            `json:"session_key"`
	Key            SessionKey        `json:"key"`
	TranscriptPath string            `json:"transcript_path"`
	MessageCount   int               `json:"message_count"`
	LastMessageAt  int64             `json:"last_message_at"`
	CreatedAt      int64             `json:"created_at"`
	UpdatedAt      int64             `json:"updated_at"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type Store interface {
	Open(ctx context.Context, key SessionKey, metadata map[string]string) (*Envelope, error)
	Get(ctx context.Context, sessionID string) (*Envelope, error)
	Append(ctx context.Context, sessionID string, message Message) (*Envelope, error)
	Recent(ctx context.Context, sessionID string, limit int) ([]Message, error)
	List(ctx context.Context, limit int) ([]*Envelope, error)
}

func copyEnvelope(env *Envelope) *Envelope {
	if env == nil {
		return nil
	}
	cp := *env
	cp.Metadata = copyMap(env.Metadata)
	return &cp
}

func copyMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func sortEnvelopes(items []*Envelope) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt == items[j].UpdatedAt {
			return items[i].SessionID < items[j].SessionID
		}
		return items[i].UpdatedAt > items[j].UpdatedAt
	})
}

func messageID(sessionID string, createdAt int64, role string) string {
	return fmt.Sprintf("%s-%d-%s", sessionID, createdAt, role)
}
