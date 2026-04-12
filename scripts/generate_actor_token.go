package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
)

func main() {
	var (
		secret      string
		workspaceID string
		actorID     string
		role        string
		scopes      string
		ttlSeconds  int
	)

	flag.StringVar(&secret, "secret", "", "actor token signing secret")
	flag.StringVar(&workspaceID, "workspace-id", authz.DefaultWorkspaceID, "workspace id")
	flag.StringVar(&actorID, "actor-id", "verify-script", "actor id")
	flag.StringVar(&role, "role", string(authz.RoleAdmin), "actor role")
	flag.StringVar(&scopes, "scopes", "*", "comma-separated scopes")
	flag.IntVar(&ttlSeconds, "ttl-seconds", 300, "token ttl in seconds")
	flag.Parse()

	if strings.TrimSpace(secret) == "" {
		secret = firstNonEmptyEnv("HTTP_SHARED_SECRET", "GATEWAY_SHARED_SECRET")
	}
	if strings.TrimSpace(secret) == "" {
		fmt.Fprintln(os.Stderr, "missing signing secret: provide --secret or HTTP_SHARED_SECRET/GATEWAY_SHARED_SECRET")
		os.Exit(1)
	}
	if strings.TrimSpace(actorID) == "" {
		fmt.Fprintln(os.Stderr, "actor-id is required")
		os.Exit(1)
	}
	if ttlSeconds <= 0 {
		fmt.Fprintln(os.Stderr, "ttl-seconds must be > 0")
		os.Exit(1)
	}

	token, err := authz.SignActorToken(strings.TrimSpace(secret), authz.ActorTokenClaims{
		WorkspaceID: strings.TrimSpace(workspaceID),
		ActorID:     strings.TrimSpace(actorID),
		Role:        authz.Role(strings.TrimSpace(role)),
		Scopes:      splitCSV(scopes),
		ExpiresAt:   time.Now().Add(time.Duration(ttlSeconds) * time.Second).Unix(),
		Nonce:       fmt.Sprintf("%d", time.Now().UnixNano()),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign actor token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(token)
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{"*"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}
