package gateway

import (
	"testing"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/node"
)

func TestAuthorizeTokenSeparatesControlAndNodeSecrets(t *testing.T) {
	oldGatewaySecret := conf.RuntimeConfInfo.Gateway.SharedSecret
	oldNodeToken := conf.RuntimeConfInfo.Nodes.PairingToken
	conf.RuntimeConfInfo.Gateway.SharedSecret = "control-secret"
	conf.RuntimeConfInfo.Nodes.PairingToken = "node-secret"
	defer func() {
		conf.RuntimeConfInfo.Gateway.SharedSecret = oldGatewaySecret
		conf.RuntimeConfInfo.Nodes.PairingToken = oldNodeToken
	}()

	service := &Service{
		cfg: &conf.RuntimeConfig{
			Gateway: conf.GatewayConf{SharedSecret: "control-secret"},
			Nodes:   conf.NodeRuntimeConf{PairingToken: "node-secret"},
		},
	}
	if !service.authorizeControlToken("control-secret") {
		t.Fatalf("expected control secret to authorize control websocket")
	}
	if service.authorizeControlToken("node-secret") {
		t.Fatalf("expected node pairing token to be rejected by control websocket")
	}
	if !service.authorizeNodeToken("node-secret") {
		t.Fatalf("expected node pairing token to authorize node websocket")
	}
	if service.authorizeNodeToken("control-secret") {
		t.Fatalf("expected control secret to be rejected by node websocket")
	}
}

func TestNormalizeControlExecRequestRequiresApproval(t *testing.T) {
	oldPrivileged := conf.BaseConfInfo.PrivilegedUserIds
	conf.BaseConfInfo.PrivilegedUserIds = map[string]bool{"owner-user": true}
	defer func() {
		conf.BaseConfInfo.PrivilegedUserIds = oldPrivileged
	}()

	req := node.NodeCommandRequest{Capability: "input.keyboard.type", UserID: "regular-user"}
	normalizeControlExecRequest(&req)
	if !req.RequireApproval {
		t.Fatalf("expected regular control exec to require approval")
	}

	privilegedReq := node.NodeCommandRequest{Capability: "input.keyboard.type", UserID: "owner-user"}
	normalizeControlExecRequest(&privilegedReq)
	if privilegedReq.RequireApproval {
		t.Fatalf("expected privileged control exec to keep approval bypass")
	}
}
