package controller

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
)

func TestGetRequestAttachesManagementHeaders(t *testing.T) {
	t.Setenv("GATEWAY_SHARED_SECRET", "shared-secret")

	ctx := context.WithValue(context.Background(), "log_id", "log-1")
	ctx = withBotActingUser(ctx, "-42")

	req := GetRequest(ctx, http.MethodGet, "http://example.com/api", nil)
	authHeader := req.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		t.Fatalf("expected bearer auth header, got %q", authHeader)
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if got := req.Header.Get(botActorTokenHeader); got != token {
		t.Fatalf("expected actor token header to match bearer token")
	}
	principal, err := authz.VerifyActorToken("shared-secret", token, time.Now())
	if err != nil {
		t.Fatalf("verify actor token: %v", err)
	}
	if principal.ActorID != "-42" {
		t.Fatalf("expected signed acting user in token, got %+v", principal)
	}
	if got := req.Header.Get("LogId"); got != "log-1" {
		t.Fatalf("expected log id header, got %q", got)
	}
}

func TestToSignedAdminActorID(t *testing.T) {
	if got := toSignedAdminActorID(7); got != "-7" {
		t.Fatalf("expected signed actor id, got %q", got)
	}
	if got := toSignedAdminActorID("-11"); got != "-11" {
		t.Fatalf("expected existing signed actor id to be preserved, got %q", got)
	}
	if got := toSignedAdminActorID("15"); got != "-15" {
		t.Fatalf("expected string actor id to be normalized, got %q", got)
	}
}
