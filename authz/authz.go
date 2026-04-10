package authz

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	DefaultWorkspaceID = "default"

	RoleOwner    Role = "owner"
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

type Role string

type Principal struct {
	WorkspaceID string   `json:"workspace_id"`
	ActorID     string   `json:"actor_id"`
	Role        Role     `json:"role"`
	Scopes      []string `json:"scopes,omitempty"`
}

type contextKey struct{}

var (
	ErrMissingPrincipal = errors.New("authz principal is required")
	ErrForbidden        = errors.New("forbidden")
)

func NormalizeWorkspaceID(workspaceID string) string {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return DefaultWorkspaceID
	}
	return workspaceID
}

func NormalizeRole(role Role) Role {
	switch Role(strings.ToLower(strings.TrimSpace(string(role)))) {
	case RoleOwner:
		return RoleOwner
	case RoleAdmin:
		return RoleAdmin
	case RoleOperator:
		return RoleOperator
	case RoleViewer:
		return RoleViewer
	default:
		return RoleViewer
	}
}

func NewPrincipal(workspaceID, actorID string, role Role, scopes []string) Principal {
	return Principal{
		WorkspaceID: NormalizeWorkspaceID(workspaceID),
		ActorID:     strings.TrimSpace(actorID),
		Role:        NormalizeRole(role),
		Scopes:      NormalizeScopes(scopes),
	}
}

func NormalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(scopes))
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(strings.ToLower(scope))
		if scope == "" || seen[scope] {
			continue
		}
		seen[scope] = true
		out = append(out, scope)
	}
	sort.Strings(out)
	return out
}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, contextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}
	principal, ok := ctx.Value(contextKey{}).(Principal)
	if !ok {
		return Principal{}, false
	}
	principal.WorkspaceID = NormalizeWorkspaceID(principal.WorkspaceID)
	principal.Role = NormalizeRole(principal.Role)
	principal.Scopes = NormalizeScopes(principal.Scopes)
	return principal, true
}

func RequirePrincipal(ctx context.Context) (Principal, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok || strings.TrimSpace(principal.ActorID) == "" {
		return Principal{}, ErrMissingPrincipal
	}
	return principal, nil
}

func WorkspaceIDFromContext(ctx context.Context) string {
	if principal, ok := PrincipalFromContext(ctx); ok {
		return principal.WorkspaceID
	}
	return DefaultWorkspaceID
}

func (p Principal) CanManageWorkspace() bool {
	switch NormalizeRole(p.Role) {
	case RoleOwner, RoleAdmin:
		return true
	default:
		return false
	}
}

func (p Principal) CanOperate() bool {
	switch NormalizeRole(p.Role) {
	case RoleOwner, RoleAdmin, RoleOperator:
		return true
	default:
		return false
	}
}

func (p Principal) CanView() bool {
	return strings.TrimSpace(p.ActorID) != "" || NormalizeRole(p.Role) == RoleViewer
}

func (p Principal) HasScope(scope string) bool {
	scope = strings.TrimSpace(strings.ToLower(scope))
	if scope == "" {
		return false
	}
	for _, item := range p.Scopes {
		if item == scope || item == "*" {
			return true
		}
	}
	return false
}

type ActorTokenClaims struct {
	WorkspaceID string   `json:"workspace_id"`
	ActorID     string   `json:"actor_id"`
	Role        Role     `json:"role"`
	Scopes      []string `json:"scopes,omitempty"`
	ExpiresAt   int64    `json:"exp"`
	Nonce       string   `json:"nonce"`
}

func SignActorToken(secret string, claims ActorTokenClaims) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", errors.New("actor token secret is required")
	}
	claims.WorkspaceID = NormalizeWorkspaceID(claims.WorkspaceID)
	claims.ActorID = strings.TrimSpace(claims.ActorID)
	claims.Role = NormalizeRole(claims.Role)
	claims.Scopes = NormalizeScopes(claims.Scopes)
	claims.Nonce = strings.TrimSpace(claims.Nonce)
	if claims.ActorID == "" {
		return "", errors.New("actor_id is required")
	}
	if claims.ExpiresAt == 0 {
		claims.ExpiresAt = time.Now().Add(10 * time.Minute).Unix()
	}
	if claims.Nonce == "" {
		claims.Nonce = hex.EncodeToString(signBytes([]byte(fmt.Sprintf("%s:%d", claims.ActorID, time.Now().UnixNano())), []byte(secret))[:8])
	}

	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	sig := signString(payload, secret)
	return payload + "." + sig, nil
}

func VerifyActorToken(secret, token string, now time.Time) (Principal, error) {
	secret = strings.TrimSpace(secret)
	token = strings.TrimSpace(token)
	if secret == "" {
		return Principal{}, errors.New("actor token secret is required")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Principal{}, errors.New("invalid actor token")
	}
	expected := signString(parts[0], secret)
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return Principal{}, errors.New("invalid actor token signature")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Principal{}, err
	}
	var claims ActorTokenClaims
	if err = json.Unmarshal(body, &claims); err != nil {
		return Principal{}, err
	}
	if now.IsZero() {
		now = time.Now()
	}
	if claims.ExpiresAt <= now.Unix() {
		return Principal{}, errors.New("actor token expired")
	}
	return NewPrincipal(claims.WorkspaceID, claims.ActorID, claims.Role, claims.Scopes), nil
}

func signString(payload, secret string) string {
	return base64.RawURLEncoding.EncodeToString(signBytes([]byte(payload), []byte(secret)))
}

func signBytes(payload, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}
