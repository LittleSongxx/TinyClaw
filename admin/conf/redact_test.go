package conf

import (
	"strings"
	"testing"
)

func TestMaskCommandSecrets(t *testing.T) {
	command := "-bot_name=TinyClawLark\n-lark_app_secret=super-secret-value\n-aliyun_token=sk-1234567890\n-http_host=:36060\n-key_file=/tmp/key.pem"

	masked := MaskCommandSecrets(command)

	if masked == command {
		t.Fatalf("expected command to be masked")
	}
	if strings.Contains(masked, "super-secret-value") {
		t.Fatalf("expected app secret to be masked")
	}
	if strings.Contains(masked, "sk-1234567890") {
		t.Fatalf("expected token to be masked")
	}
	if !strings.Contains(masked, "-bot_name=TinyClawLark") {
		t.Fatalf("expected non-sensitive fields to stay visible")
	}
	if !strings.Contains(masked, "-key_file=/tmp/key.pem") {
		t.Fatalf("expected *_file fields to stay unchanged")
	}
}

func TestMergeMaskedCommand(t *testing.T) {
	stored := "-bot_name=TinyClawLark\n-lark_app_secret=super-secret-value\n-aliyun_token=sk-1234567890\n-http_host=:36060"
	incoming := MaskCommandSecrets(stored)

	merged := MergeMaskedCommand(incoming, stored)

	if merged != stored {
		t.Fatalf("expected masked command to merge back to stored raw command")
	}
}

func TestMergeMaskedStoredSecret(t *testing.T) {
	stored := "-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----"
	masked := MaskStoredSecret("key_file", stored)

	if got := MergeMaskedStoredSecret("key_file", masked, stored); got != stored {
		t.Fatalf("expected masked key file to merge back to stored value")
	}
}
