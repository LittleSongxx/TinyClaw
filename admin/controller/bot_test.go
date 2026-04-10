package controller

import (
	"context"
	"net/http"
	"testing"
)

func TestGetRequestAttachesManagementHeaders(t *testing.T) {
	t.Setenv("GATEWAY_SHARED_SECRET", "shared-secret")

	ctx := context.WithValue(context.Background(), "log_id", "log-1")
	ctx = withBotActingUser(ctx, "-42")

	req := GetRequest(ctx, http.MethodGet, "http://example.com/api", nil)
	if got := req.Header.Get("Authorization"); got != "Bearer shared-secret" {
		t.Fatalf("expected bearer auth header, got %q", got)
	}
	if got := req.Header.Get(botManagementTokenHeader); got != "shared-secret" {
		t.Fatalf("expected management token header, got %q", got)
	}
	if got := req.Header.Get(botActingUserHeader); got != "-42" {
		t.Fatalf("expected acting user header, got %q", got)
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
