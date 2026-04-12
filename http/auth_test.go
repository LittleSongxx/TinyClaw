package http

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/node"
)

func TestIsTrustedManagementRequestUsesActorToken(t *testing.T) {
	oldSecret := conf.RuntimeConfInfo.Gateway.SharedSecret
	conf.RuntimeConfInfo.Gateway.SharedSecret = "shared-secret-with-enough-length-123"
	defer func() {
		conf.RuntimeConfInfo.Gateway.SharedSecret = oldSecret
	}()
	token, err := authz.SignActorToken(conf.RuntimeConfInfo.Gateway.SharedSecret, authz.ActorTokenClaims{
		WorkspaceID: "default",
		ActorID:     "owner-user",
		Role:        authz.RoleOwner,
		ExpiresAt:   time.Now().Add(time.Minute).Unix(),
		Nonce:       "nonce",
	})
	if err != nil {
		t.Fatalf("sign actor token: %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.com/conf/get", nil)
	req.RemoteAddr = "203.0.113.7:3456"
	req.Header.Set(actorTokenHeader, token)

	principal, ok := principalFromManagementRequest(req)
	if !ok || principal.ActorID != "owner-user" {
		t.Fatalf("expected actor token to authorize management request, got %+v ok=%v", principal, ok)
	}
}

func TestIsTrustedManagementRequestRejectsLoopbackWithoutActorToken(t *testing.T) {
	oldSecret := conf.RuntimeConfInfo.Gateway.SharedSecret
	conf.RuntimeConfInfo.Gateway.SharedSecret = "shared-secret-with-enough-length-123"
	defer func() {
		conf.RuntimeConfInfo.Gateway.SharedSecret = oldSecret
	}()

	loopbackReq := httptest.NewRequest("GET", "http://example.com/conf/get", nil)
	loopbackReq.RemoteAddr = "127.0.0.1:9999"
	if isTrustedManagementRequest(loopbackReq) {
		t.Fatalf("expected loopback request without actor token to be rejected")
	}

	remoteReq := httptest.NewRequest("GET", "http://example.com/conf/get", nil)
	remoteReq.RemoteAddr = "203.0.113.7:3456"
	if isTrustedManagementRequest(remoteReq) {
		t.Fatalf("expected remote request without token to be rejected")
	}
}

func TestIsTrustedManagementRequestRejectsPlainSharedSecretBearer(t *testing.T) {
	oldSecret := conf.RuntimeConfInfo.Gateway.SharedSecret
	conf.RuntimeConfInfo.Gateway.SharedSecret = "shared-secret-with-enough-length-123"
	defer func() {
		conf.RuntimeConfInfo.Gateway.SharedSecret = oldSecret
	}()

	req := httptest.NewRequest("GET", "http://example.com/conf/get", nil)
	req.RemoteAddr = "203.0.113.7:3456"
	req.Header.Set("Authorization", "Bearer "+conf.RuntimeConfInfo.Gateway.SharedSecret)

	if isTrustedManagementRequest(req) {
		t.Fatalf("expected plain shared secret bearer to be rejected")
	}
}

func TestActingUserIDFromRequestRequiresTrustedRequest(t *testing.T) {
	trustedReq := httptest.NewRequest("GET", "http://example.com/communicate?user_id=query-user", nil)
	trustedReq = trustedReq.WithContext(authz.WithPrincipal(trustedReq.Context(), authz.NewPrincipal("default", "token-user", authz.RoleAdmin, nil)))
	if got := actingUserIDFromRequest(trustedReq); got != "token-user" {
		t.Fatalf("expected acting user from principal, got %q", got)
	}

	untrustedReq := httptest.NewRequest("GET", "http://example.com/communicate?user_id=query-user", nil)
	untrustedReq.RemoteAddr = "203.0.113.7:3456"
	if got := actingUserIDFromRequest(untrustedReq); got != "" {
		t.Fatalf("expected acting user to be ignored for untrusted request, got %q", got)
	}
}

func TestNormalizeManagedNodeCommandRequestForcesApprovalForRegularUsers(t *testing.T) {
	req := httptest.NewRequest("POST", "http://example.com/gateway/node/command", nil)
	req = req.WithContext(authz.WithPrincipal(req.Context(), authz.NewPrincipal("default", "regular-user", authz.RoleOperator, nil)))

	cmd := node.NodeCommandRequest{Capability: "system.exec"}
	normalizeManagedNodeCommandRequest(req, &cmd)
	if cmd.UserID != "regular-user" {
		t.Fatalf("expected acting user to be copied onto command, got %q", cmd.UserID)
	}
	if !cmd.RequireApproval {
		t.Fatalf("expected regular management command to require approval")
	}

	privilegedReq := httptest.NewRequest("POST", "http://example.com/gateway/node/command", nil)
	privilegedReq = privilegedReq.WithContext(authz.WithPrincipal(privilegedReq.Context(), authz.NewPrincipal("default", "owner-user", authz.RoleOwner, []string{"*"})))

	privilegedCmd := node.NodeCommandRequest{Capability: "system.exec"}
	normalizeManagedNodeCommandRequest(privilegedReq, &privilegedCmd)
	if !privilegedCmd.RequireApproval {
		t.Fatalf("expected owner management actor to still require policy/approval binding")
	}
}
