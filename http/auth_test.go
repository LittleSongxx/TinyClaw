package http

import (
	"net/http/httptest"
	"testing"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/node"
)

func TestIsTrustedManagementRequestUsesSharedSecret(t *testing.T) {
	oldSecret := conf.RuntimeConfInfo.Gateway.SharedSecret
	conf.RuntimeConfInfo.Gateway.SharedSecret = "shared-secret"
	defer func() {
		conf.RuntimeConfInfo.Gateway.SharedSecret = oldSecret
	}()

	req := httptest.NewRequest("GET", "http://example.com/conf/get", nil)
	req.RemoteAddr = "203.0.113.7:3456"
	req.Header.Set("Authorization", "Bearer shared-secret")

	if !isTrustedManagementRequest(req) {
		t.Fatalf("expected bearer token to authorize management request")
	}
}

func TestIsTrustedManagementRequestFallsBackToLoopback(t *testing.T) {
	oldSecret := conf.RuntimeConfInfo.Gateway.SharedSecret
	conf.RuntimeConfInfo.Gateway.SharedSecret = ""
	defer func() {
		conf.RuntimeConfInfo.Gateway.SharedSecret = oldSecret
	}()

	loopbackReq := httptest.NewRequest("GET", "http://example.com/conf/get", nil)
	loopbackReq.RemoteAddr = "127.0.0.1:9999"
	if !isTrustedManagementRequest(loopbackReq) {
		t.Fatalf("expected loopback request to be trusted when no shared secret is configured")
	}

	remoteReq := httptest.NewRequest("GET", "http://example.com/conf/get", nil)
	remoteReq.RemoteAddr = "203.0.113.7:3456"
	if isTrustedManagementRequest(remoteReq) {
		t.Fatalf("expected remote request without token to be rejected")
	}
}

func TestActingUserIDFromRequestRequiresTrustedRequest(t *testing.T) {
	oldSecret := conf.RuntimeConfInfo.Gateway.SharedSecret
	conf.RuntimeConfInfo.Gateway.SharedSecret = "shared-secret"
	defer func() {
		conf.RuntimeConfInfo.Gateway.SharedSecret = oldSecret
	}()

	trustedReq := httptest.NewRequest("GET", "http://example.com/communicate?user_id=query-user", nil)
	trustedReq.RemoteAddr = "203.0.113.7:3456"
	trustedReq.Header.Set(managementTokenHeader, "shared-secret")
	trustedReq.Header.Set(actingUserHeader, "header-user")
	if got := actingUserIDFromRequest(trustedReq); got != "header-user" {
		t.Fatalf("expected acting user from trusted header, got %q", got)
	}

	untrustedReq := httptest.NewRequest("GET", "http://example.com/communicate?user_id=query-user", nil)
	untrustedReq.RemoteAddr = "203.0.113.7:3456"
	if got := actingUserIDFromRequest(untrustedReq); got != "" {
		t.Fatalf("expected acting user to be ignored for untrusted request, got %q", got)
	}
}

func TestNormalizeManagedNodeCommandRequestForcesApprovalForRegularUsers(t *testing.T) {
	oldSecret := conf.RuntimeConfInfo.Gateway.SharedSecret
	oldPrivileged := conf.BaseConfInfo.PrivilegedUserIds
	conf.RuntimeConfInfo.Gateway.SharedSecret = "shared-secret"
	conf.BaseConfInfo.PrivilegedUserIds = map[string]bool{"owner-user": true}
	defer func() {
		conf.RuntimeConfInfo.Gateway.SharedSecret = oldSecret
		conf.BaseConfInfo.PrivilegedUserIds = oldPrivileged
	}()

	req := httptest.NewRequest("POST", "http://example.com/gateway/node/command", nil)
	req.RemoteAddr = "203.0.113.7:3456"
	req.Header.Set(managementTokenHeader, "shared-secret")
	req.Header.Set(actingUserHeader, "regular-user")

	cmd := node.NodeCommandRequest{Capability: "system.exec"}
	normalizeManagedNodeCommandRequest(req, &cmd)
	if cmd.UserID != "regular-user" {
		t.Fatalf("expected acting user to be copied onto command, got %q", cmd.UserID)
	}
	if !cmd.RequireApproval {
		t.Fatalf("expected regular management command to require approval")
	}

	privilegedReq := httptest.NewRequest("POST", "http://example.com/gateway/node/command", nil)
	privilegedReq.RemoteAddr = "203.0.113.7:3456"
	privilegedReq.Header.Set(managementTokenHeader, "shared-secret")
	privilegedReq.Header.Set(actingUserHeader, "owner-user")

	privilegedCmd := node.NodeCommandRequest{Capability: "system.exec"}
	normalizeManagedNodeCommandRequest(privilegedReq, &privilegedCmd)
	if privilegedCmd.RequireApproval {
		t.Fatalf("expected privileged management actor to keep approval bypass")
	}
}
