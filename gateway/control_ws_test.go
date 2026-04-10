package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/node"
)

func TestAuthorizeControlConnectUsesActorTokenAndRejectsNodeToken(t *testing.T) {
	oldGatewaySecret := conf.RuntimeConfInfo.Gateway.SharedSecret
	conf.RuntimeConfInfo.Gateway.SharedSecret = "control-secret"
	defer func() {
		conf.RuntimeConfInfo.Gateway.SharedSecret = oldGatewaySecret
	}()
	token, err := authz.SignActorToken("control-secret", authz.ActorTokenClaims{
		WorkspaceID: "default",
		ActorID:     "owner-user",
		Role:        authz.RoleOwner,
		ExpiresAt:   time.Now().Add(time.Minute).Unix(),
		Nonce:       "nonce",
	})
	if err != nil {
		t.Fatalf("sign actor token: %v", err)
	}

	service := &Service{
		cfg: &conf.RuntimeConfig{
			Gateway: conf.GatewayConf{SharedSecret: "control-secret"},
		},
	}
	if _, ok := service.authorizeControlConnect(ConnectFrame{
		Type:            FrameTypeConnect,
		ProtocolVersion: ProtocolVersionV1,
		Role:            "control",
		Auth:            AuthInfo{Type: "actor_hmac", ActorToken: token},
	}); !ok {
		t.Fatalf("expected actor token to authorize control websocket")
	}
	if service.authorizeNodeToken("node-secret") {
		t.Fatalf("expected static node pairing token to be rejected")
	}
}

func TestNormalizeControlExecRequestRequiresApproval(t *testing.T) {
	ctx := authz.WithPrincipal(context.Background(), authz.NewPrincipal("default", "owner-user", authz.RoleOwner, []string{"*"}))

	req := node.NodeCommandRequest{Capability: "input.keyboard.type", UserID: "regular-user"}
	normalizeControlExecRequest(ctx, &req)
	if !req.RequireApproval {
		t.Fatalf("expected control exec to require approval")
	}
	if req.ActorRole != string(authz.RoleOwner) || req.WorkspaceID != "default" {
		t.Fatalf("expected principal fields to be copied, got %+v", req)
	}
}
