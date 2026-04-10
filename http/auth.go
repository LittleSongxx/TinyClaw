package http

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/utils"
)

const (
	actorTokenHeader = "X-TinyClaw-Actor-Token"
)

func withManagementAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromManagementRequest(r)
		if ok {
			next(w, r.WithContext(authz.WithPrincipal(r.Context(), principal)))
			return
		}

		utils.Failure(r.Context(), w, r, param.CodeNotLogin, "management authorization required", nil)
	}
}

func isTrustedManagementRequest(r *http.Request) bool {
	_, ok := principalFromManagementRequest(r)
	return ok
}

func principalFromManagementRequest(r *http.Request) (authz.Principal, bool) {
	if r == nil {
		return authz.Principal{}, false
	}
	if hasVerifiedClientCertificate(r) {
		return managementPrincipal(r, "mtls-admin"), true
	}
	secret := managementSigningSecret()
	if secret != "" {
		if token := strings.TrimSpace(readActorToken(r)); token != "" {
			principal, err := authz.VerifyActorToken(secret, token, time.Now())
			return principal, err == nil
		}
	}
	return authz.Principal{}, false
}

func readActorToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	if token := strings.TrimSpace(r.Header.Get(actorTokenHeader)); token != "" {
		return token
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		token := strings.TrimSpace(authHeader[7:])
		if strings.Contains(token, ".") {
			return token
		}
	}
	return ""
}

func managementSigningSecret() string {
	if secret := strings.TrimSpace(os.Getenv("HTTP_SHARED_SECRET")); secret != "" {
		return secret
	}
	return strings.TrimSpace(conf.RuntimeConfInfo.Gateway.SharedSecret)
}

func hasVerifiedClientCertificate(r *http.Request) bool {
	return r != nil && r.TLS != nil && (len(r.TLS.VerifiedChains) > 0 || len(r.TLS.PeerCertificates) > 0)
}

func actingUserIDFromRequest(r *http.Request) string {
	if principal, ok := authz.PrincipalFromContext(r.Context()); ok {
		return principal.ActorID
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func managementPrincipal(r *http.Request, defaultActor string) authz.Principal {
	workspaceID := authz.NormalizeWorkspaceID(firstNonEmpty(
		r.URL.Query().Get("workspace_id"),
		r.URL.Query().Get("workspace"),
		r.Header.Get("X-TinyClaw-Workspace"),
	))
	actorID := strings.TrimSpace(firstNonEmpty(
		r.URL.Query().Get("actor_id"),
	))
	if actorID == "" {
		actorID = defaultActor
	}
	return authz.NewPrincipal(workspaceID, actorID, authz.RoleAdmin, []string{"*"})
}
