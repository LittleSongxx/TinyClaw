package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/LittleSongxx/TinyClaw/agent"
	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/plugins"
	"github.com/LittleSongxx/TinyClaw/session"
	"github.com/LittleSongxx/TinyClaw/tooling"
	"github.com/google/uuid"
)

type ChannelAdapter interface {
	Name() string
	ResolveSessionKey(message InboundMessage) session.SessionKey
}

type ApprovalStore interface {
	Save(ctx context.Context, decision node.ApprovalDecision) error
	Get(ctx context.Context, commandID string) (*node.ApprovalDecision, error)
}

type InboundMessage struct {
	Channel   string            `json:"channel"`
	AccountID string            `json:"account_id"`
	PeerID    string            `json:"peer_id"`
	GroupID   string            `json:"group_id"`
	ThreadID  string            `json:"thread_id"`
	MessageID string            `json:"message_id"`
	Kind      string            `json:"kind"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type Service struct {
	cfg       *conf.RuntimeConfig
	sessions  session.Store
	nodes     *node.Manager
	tools     *tooling.Broker
	agent     *agent.Runtime
	approvals ApprovalStore
	idempotency *idempotencyCache
	plugins   *plugins.Registry

	mu       sync.RWMutex
	adapters map[string]ChannelAdapter
}

type memoryApprovalStore struct {
	mu    sync.RWMutex
	items map[string]node.ApprovalDecision
}

var (
	defaultService *Service
	defaultOnce    sync.Once
)

func Init() *Service {
	defaultOnce.Do(func() {
		if conf.RuntimeConfInfo.Sessions.TranscriptDir == "" {
			conf.InitRuntimeConf()
		}

		store := session.InitDefaultStore(conf.RuntimeConfInfo.Sessions.TranscriptDir)
		nodes := node.NewManager()
		providers := []tooling.ToolProvider{tooling.NewNodeProvider(nodes)}
		if conf.FeatureConfInfo.LegacyTaskToolsEnabled() {
			providers = append([]tooling.ToolProvider{tooling.NewRegistryProvider(tooling.NewRegistryFromTaskTools())}, providers...)
		}
		tools := tooling.NewBroker(providers...)
		runtime := agent.NewRuntime(
			&agent.SessionAssembler{Store: store, Limit: conf.RuntimeConfInfo.Sessions.ContextWindow},
			nil,
			tools,
			nodes,
		)

		defaultService = NewService(conf.RuntimeConfInfo, store, tools, nodes, runtime, newMemoryApprovalStore())
	})
	return defaultService
}

func DefaultService() *Service {
	if defaultService == nil {
		return Init()
	}
	return defaultService
}

func NewService(cfg *conf.RuntimeConfig, sessions session.Store, tools *tooling.Broker, nodes *node.Manager, runtime *agent.Runtime, approvals ApprovalStore) *Service {
	if approvals == nil {
		approvals = newMemoryApprovalStore()
	}
	service := &Service{
		cfg:       cfg,
		sessions:  sessions,
		tools:     tools,
		nodes:     nodes,
		agent:     runtime,
		approvals: approvals,
		idempotency: newIdempotencyCache(5 * time.Minute),
		plugins:   plugins.NewDefaultRegistry(),
		adapters:  make(map[string]ChannelAdapter),
	}
	if nodes != nil {
		nodes.SetEventObserver(service.recordActionEvent)
	}
	return service
}

func (s *Service) PluginRegistry() *plugins.Registry {
	if s == nil {
		return nil
	}
	if s.plugins == nil {
		s.plugins = plugins.NewDefaultRegistry()
	}
	return s.plugins
}

func (s *Service) RegisterAdapter(adapter ChannelAdapter) {
	if s == nil || adapter == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adapters[adapter.Name()] = adapter
}

func (s *Service) BeginInbound(ctx context.Context, message InboundMessage) (*session.Envelope, *param.ContextState, error) {
	if s == nil || s.sessions == nil {
		return nil, nil, errors.New("gateway sessions are not initialized")
	}

	key := s.resolveSessionKey(message)
	env, err := s.sessions.Open(ctx, key, message.Metadata)
	if err != nil {
		return nil, nil, err
	}

	state := &param.ContextState{
		UseRecord:    false,
		WorkspaceID:  key.WorkspaceID,
		SessionID:    env.SessionID,
		SessionKey:   env.SessionKey,
		SessionScope: key.Scope(),
		Channel:      key.Channel,
		AccountID:    key.AccountID,
		PeerID:       key.PeerID,
		GroupID:      key.GroupID,
		ThreadID:     key.ThreadID,
		Source:       firstNonEmpty(message.Kind, key.Scope()),
	}
	return env, state, nil
}

func (s *Service) ListNodes(ctx context.Context) []node.NodeDescriptor {
	if s == nil || s.nodes == nil {
		return nil
	}
	return s.nodes.ListNodes(ctx)
}

func (s *Service) ExecuteNodeCommand(ctx context.Context, req node.NodeCommandRequest) (*node.NodeCommandResult, error) {
	if s == nil || s.nodes == nil {
		return nil, errors.New("gateway node broker is not initialized")
	}
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	if req.TimeoutSec <= 0 && s.cfg != nil {
		req.TimeoutSec = s.cfg.Nodes.DefaultCommandTimeoutSec
	}
	return s.nodes.Execute(ctx, req)
}

func (s *Service) ListSessions(ctx context.Context, limit int) ([]*session.Envelope, error) {
	if s == nil || s.sessions == nil {
		return nil, nil
	}
	return s.sessions.List(ctx, limit)
}

func (s *Service) ListApprovals(ctx context.Context) []node.ApprovalRequest {
	if s == nil || s.nodes == nil {
		return nil
	}
	return s.nodes.ListApprovals(ctx)
}

func (s *Service) DecideApproval(ctx context.Context, decision node.ApprovalDecision) (map[string]interface{}, error) {
	if s == nil || s.nodes == nil {
		return nil, errors.New("gateway approval broker is not initialized")
	}
	approval, result, err := s.nodes.DecideApproval(ctx, decision)
	if err != nil {
		return nil, err
	}
	if s.approvals != nil {
		if decision.CreatedAt == 0 {
			decision.CreatedAt = approval.CreatedAt
		}
		if decision.NodeID == "" && approval != nil {
			decision.NodeID = approval.NodeID
		}
		if decision.SessionID == "" && approval != nil {
			decision.SessionID = approval.SessionID
		}
		_ = s.approvals.Save(ctx, decision)
	}
	return map[string]interface{}{
		"approval": approval,
		"result":   result,
	}, nil
}

func (s *Service) ToolBroker() *tooling.Broker {
	if s == nil {
		return nil
	}
	return s.tools
}

func (s *Service) recordActionEvent(event node.ActionEvent) {
	if s == nil {
		return
	}
	if event.CreatedAt == 0 {
		event.CreatedAt = time.Now().Unix()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		logger.Warn("marshal node action event fail", "err", err, "event_type", event.Type)
		return
	}

	logger.Info("node action event",
		"type", event.Type,
		"action_id", event.ActionID,
		"approval_id", event.ApprovalID,
		"session_id", event.SessionID,
		"user_id", event.UserID,
		"node_id", event.NodeID,
		"capability", event.Capability,
		"success", event.Success,
		"detail", event.Detail,
	)

	if s.sessions == nil || event.SessionID == "" {
		return
	}

	metadata := map[string]string{
		"kind":       "node_action_event",
		"event_type": event.Type,
		"action_id":  event.ActionID,
		"node_id":    event.NodeID,
		"capability": event.Capability,
	}
	if event.ApprovalID != "" {
		metadata["approval_id"] = event.ApprovalID
	}
	if event.Mode != "" {
		metadata["approval_mode"] = string(event.Mode)
	}
	if event.Detail != "" {
		metadata["detail"] = event.Detail
	}
	if _, appendErr := s.sessions.Append(context.Background(), event.SessionID, session.Message{
		Role:      session.RoleSystem,
		Content:   string(payload),
		CreatedAt: event.CreatedAt,
		Metadata:  metadata,
	}); appendErr != nil {
		logger.Warn("append node action event fail", "err", appendErr, "session_id", event.SessionID, "event_type", event.Type)
	}
}

func (s *Service) ListSessionMeta(limit int) ([]db.SessionMeta, error) {
	return db.ListSessionMeta(limit)
}

func (s *Service) ListSessionMetaInWorkspace(ctx context.Context, limit int) ([]db.SessionMeta, error) {
	return db.ListSessionMetaInWorkspace(authz.WorkspaceIDFromContext(ctx), limit)
}

func (s *Service) resolveSessionKey(message InboundMessage) session.SessionKey {
	s.mu.RLock()
	adapter := s.adapters[message.Channel]
	s.mu.RUnlock()
	if adapter != nil {
		key := adapter.ResolveSessionKey(message)
		key.WorkspaceID = authz.NormalizeWorkspaceID(firstNonEmpty(key.WorkspaceID, message.Metadata["workspace_id"], message.Metadata["workspace"]))
		return key
	}

	kind := message.Kind
	if kind == "" {
		if message.GroupID != "" || message.ThreadID != "" {
			kind = "group"
		} else {
			kind = "dm"
		}
	}

	key := session.SessionKey{
		WorkspaceID: authz.NormalizeWorkspaceID(firstNonEmpty(message.Metadata["workspace_id"], message.Metadata["workspace"], authz.WorkspaceIDFromContext(context.Background()))),
		Channel:   firstNonEmpty(message.Channel, "web"),
		AccountID: firstNonEmpty(message.AccountID, "default"),
		PeerID:    message.PeerID,
		GroupID:   message.GroupID,
		ThreadID:  message.ThreadID,
		Kind:      kind,
	}
	if kind == "cron" || kind == "debug" {
		key.Nonce = firstNonEmpty(message.MessageID, uuid.NewString())
	}
	return key
}

func (s *Service) authorizeControlToken(token string) bool {
	if s == nil || s.cfg == nil {
		return false
	}
	secret := strings.TrimSpace(s.cfg.Gateway.SharedSecret)
	if secret == "" || token == "" {
		return false
	}
	return token == secret
}

func (s *Service) authorizeNodeToken(token string) bool {
	return false
}

func (s *Service) authorizeControlConnect(connect ConnectFrame) (authz.Principal, bool) {
	if s == nil || s.cfg == nil {
		return authz.Principal{}, false
	}
	if connect.ProtocolVersion != ProtocolVersionV1 {
		return authz.Principal{}, false
	}
	secret := strings.TrimSpace(s.cfg.Gateway.SharedSecret)
	if secret == "" {
		return authz.Principal{}, false
	}
	token := strings.TrimSpace(firstNonEmpty(connect.Auth.ActorToken, connect.Auth.Token))
	if token == "" {
		return authz.Principal{}, false
	}
	principal, err := authz.VerifyActorToken(secret, token, time.Now())
	if err != nil {
		return authz.Principal{}, false
	}
	if connect.WorkspaceID != "" && !sameGatewayWorkspace(connect.WorkspaceID, principal.WorkspaceID) {
		return authz.Principal{}, false
	}
	return principal, true
}

func (s *Service) idempotencyKey(principal authz.Principal, method, key string) string {
	if key == "" {
		return ""
	}
	return authz.NormalizeWorkspaceID(principal.WorkspaceID) + ":" + strings.TrimSpace(principal.ActorID) + ":" + strings.TrimSpace(method) + ":" + strings.TrimSpace(key)
}

func sameGatewayWorkspace(left, right string) bool {
	return authz.NormalizeWorkspaceID(left) == authz.NormalizeWorkspaceID(right)
}

func newMemoryApprovalStore() *memoryApprovalStore {
	return &memoryApprovalStore{items: make(map[string]node.ApprovalDecision)}
}

func (m *memoryApprovalStore) Save(ctx context.Context, decision node.ApprovalDecision) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[decision.CommandID] = decision
	return nil
}

func (m *memoryApprovalStore) Get(ctx context.Context, commandID string) (*node.ApprovalDecision, error) {
	if m == nil {
		return nil, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	decision, ok := m.items[commandID]
	if !ok {
		return nil, nil
	}
	return &decision, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
