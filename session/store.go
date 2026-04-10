package session

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/google/uuid"
)

var DefaultStore Store

type FileStore struct {
	root      string
	indexFile string

	mu     sync.Mutex
	loaded bool
	byID   map[string]*Envelope
	byKey  map[string]string
}

func NewFileStore(root string) *FileStore {
	return &FileStore{
		root:      root,
		indexFile: filepath.Join(root, "sessions.json"),
		byID:      make(map[string]*Envelope),
		byKey:     make(map[string]string),
	}
}

func InitDefaultStore(root string) Store {
	DefaultStore = NewFileStore(root)
	return DefaultStore
}

func (s *FileStore) Open(ctx context.Context, key SessionKey, metadata map[string]string) (*Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key.WorkspaceID = authz.NormalizeWorkspaceID(firstNonEmpty(key.WorkspaceID, authz.WorkspaceIDFromContext(ctx)))
	if err := s.loadLocked(); err != nil {
		return nil, err
	}

	stableKey := key.StableKey()
	if sessionID, ok := s.byKey[stableKey]; ok {
		if existing, ok := s.byID[sessionID]; ok {
			if len(metadata) > 0 {
				if existing.Metadata == nil {
					existing.Metadata = make(map[string]string, len(metadata))
				}
				for k, v := range metadata {
					existing.Metadata[k] = v
				}
				existing.UpdatedAt = time.Now().Unix()
				if err := s.persistLocked(); err != nil {
					return nil, err
				}
			}
			return copyEnvelope(existing), nil
		}
	}

	now := time.Now().Unix()
	sessionID := uuid.NewString()
	env := &Envelope{
		WorkspaceID:    key.WorkspaceID,
		SessionID:      sessionID,
		SessionKey:     stableKey,
		Key:            key,
		TranscriptPath: filepath.Join(s.root, sessionID+".jsonl"),
		CreatedAt:      now,
		UpdatedAt:      now,
		Metadata:       copyMap(metadata),
	}
	s.byID[sessionID] = env
	s.byKey[stableKey] = sessionID

	if err := s.persistLocked(); err != nil {
		return nil, err
	}
	if err := s.upsertMeta(env); err != nil {
		return nil, err
	}
	return copyEnvelope(env), nil
}

func (s *FileStore) Get(ctx context.Context, sessionID string) (*Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadLocked(); err != nil {
		return nil, err
	}
	return copyEnvelope(s.byID[sessionID]), nil
}

func (s *FileStore) Append(ctx context.Context, sessionID string, message Message) (*Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadLocked(); err != nil {
		return nil, err
	}

	env, ok := s.byID[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}

	if message.CreatedAt == 0 {
		message.CreatedAt = time.Now().Unix()
	}
	if message.ID == "" {
		message.ID = messageID(sessionID, message.CreatedAt, message.Role)
	}

	file, err := os.OpenFile(env.TranscriptPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(&message); err != nil {
		return nil, err
	}

	env.MessageCount++
	env.LastMessageAt = message.CreatedAt
	env.UpdatedAt = time.Now().Unix()

	if err := s.persistLocked(); err != nil {
		return nil, err
	}
	if err := s.upsertMeta(env); err != nil {
		return nil, err
	}
	return copyEnvelope(env), nil
}

func (s *FileStore) Recent(ctx context.Context, sessionID string, limit int) ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadLocked(); err != nil {
		return nil, err
	}

	env, ok := s.byID[sessionID]
	if !ok {
		return nil, nil
	}

	file, err := os.Open(env.TranscriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	messages := make([]Message, 0, limit)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}
	return messages, nil
}

func (s *FileStore) List(ctx context.Context, limit int) ([]*Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadLocked(); err != nil {
		return nil, err
	}

	items := make([]*Envelope, 0, len(s.byID))
	for _, env := range s.byID {
		items = append(items, copyEnvelope(env))
	}
	sortEnvelopes(items)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *FileStore) loadLocked() error {
	if s.loaded {
		return nil
	}
	if err := os.MkdirAll(s.root, 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(s.indexFile)
	if err != nil {
		if os.IsNotExist(err) {
			s.loaded = true
			return nil
		}
		return err
	}

	var envelopes []*Envelope
	if len(data) > 0 {
		if err := json.Unmarshal(data, &envelopes); err != nil {
			return err
		}
	}
	for _, env := range envelopes {
		env.WorkspaceID = authz.NormalizeWorkspaceID(firstNonEmpty(env.WorkspaceID, env.Key.WorkspaceID))
		env.Key.WorkspaceID = env.WorkspaceID
		if env.SessionKey == "" {
			env.SessionKey = env.Key.StableKey()
		}
		s.byID[env.SessionID] = env
		s.byKey[env.SessionKey] = env.SessionID
	}
	s.loaded = true
	return nil
}

func (s *FileStore) persistLocked() error {
	items := make([]*Envelope, 0, len(s.byID))
	for _, env := range s.byID {
		items = append(items, env)
	}
	sortEnvelopes(items)

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.indexFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.indexFile)
}

func (s *FileStore) upsertMeta(env *Envelope) error {
	return db.UpsertSessionMeta(&db.SessionMeta{
		WorkspaceID:    env.WorkspaceID,
		SessionID:      env.SessionID,
		SessionKey:     env.SessionKey,
		Channel:        env.Key.Channel,
		AccountID:      env.Key.AccountID,
		PeerID:         env.Key.PeerID,
		GroupID:        env.Key.GroupID,
		ThreadID:       env.Key.ThreadID,
		Kind:           env.Key.Scope(),
		TranscriptPath: env.TranscriptPath,
		MessageCount:   env.MessageCount,
		LastMessageAt:  env.LastMessageAt,
		CreateTime:     env.CreatedAt,
		UpdateTime:     env.UpdatedAt,
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func RecentContext(ctx context.Context, sessionID string, limit int) ([]Message, error) {
	if DefaultStore == nil || sessionID == "" {
		return nil, nil
	}
	return DefaultStore.Recent(ctx, sessionID, limit)
}

func AppendConversation(ctx context.Context, sessionID, prompt, answer string, metadata map[string]string) error {
	if DefaultStore == nil || sessionID == "" {
		return nil
	}

	now := time.Now().Unix()
	if prompt != "" {
		if _, err := DefaultStore.Append(ctx, sessionID, Message{
			Role:      RoleUser,
			Content:   prompt,
			CreatedAt: now,
			Metadata:  copyMap(metadata),
		}); err != nil {
			return err
		}
	}
	if answer != "" {
		if _, err := DefaultStore.Append(ctx, sessionID, Message{
			Role:      RoleAssistant,
			Content:   answer,
			CreatedAt: now,
			Metadata:  copyMap(metadata),
		}); err != nil {
			return err
		}
	}
	return nil
}

func DefaultTranscriptDir() string {
	return conf.RuntimeConfInfo.Sessions.TranscriptDir
}
