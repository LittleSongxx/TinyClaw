package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LittleSongxx/TinyClaw/admin/checkpoint"
	adminConf "github.com/LittleSongxx/TinyClaw/admin/conf"
	"github.com/LittleSongxx/TinyClaw/admin/db"
	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/logger"
)

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "tinyclaw-admin-controller-*")
	if err != nil {
		panic(err)
	}
	os.Setenv("DB_TYPE", "sqlite3")
	os.Setenv("DB_CONF", filepath.Join(tempDir, "tinyclaw-admin-controller.db"))
	os.Setenv("REGISTER_TYPE", "")
	adminConf.InitConfig()
	db.InitTable()

	os.Exit(m.Run())
}

func resetBotSelectorState(t *testing.T) {
	t.Helper()
	_ = db.DeleteAllBotData()
	checkpoint.BotMap = sync.Map{}
	adminConf.RegisterConfInfo.Type = ""
}

func TestGetRequestAttachesManagementHeaders(t *testing.T) {
	t.Setenv("GATEWAY_SHARED_SECRET", "shared-secret")

	ctx := logger.WithLogID(context.Background(), "log-1")
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

func TestGetAllOnlineBotFallbackToDB(t *testing.T) {
	resetBotSelectorState(t)
	if err := db.CreateBot("http://127.0.0.1:36060", "fallback-bot", "", "", "", ""); err != nil {
		t.Fatalf("create bot: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/bot/online", nil)
	recorder := httptest.NewRecorder()

	GetAllOnlineBot(recorder, req)

	var response struct {
		Code int                    `json:"code"`
		Data []checkpoint.BotStatus `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Code != 0 {
		t.Fatalf("expected success code, got %d, body=%s", response.Code, recorder.Body.String())
	}
	if len(response.Data) != 1 {
		t.Fatalf("expected one bot from db fallback, got %d", len(response.Data))
	}
	if response.Data[0].Name != "fallback-bot" {
		t.Fatalf("expected fallback bot name, got %+v", response.Data[0])
	}
}

func TestGetAllOnlineBotPreferOnlineMap(t *testing.T) {
	resetBotSelectorState(t)
	if err := db.CreateBot("http://127.0.0.1:36061", "db-bot", "", "", "", ""); err != nil {
		t.Fatalf("create db bot: %v", err)
	}
	checkpoint.BotMap.Store(100, &checkpoint.BotStatus{
		Id:        "100",
		Name:      "online-bot",
		Address:   "http://online",
		Status:    checkpoint.OnlineStatus,
		LastCheck: time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/bot/online", nil)
	recorder := httptest.NewRecorder()

	GetAllOnlineBot(recorder, req)

	var response struct {
		Code int                    `json:"code"`
		Data []checkpoint.BotStatus `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Code != 0 {
		t.Fatalf("expected success code, got %d, body=%s", response.Code, recorder.Body.String())
	}
	if len(response.Data) != 1 {
		t.Fatalf("expected one bot from online map, got %d", len(response.Data))
	}
	if response.Data[0].Name != "online-bot" {
		t.Fatalf("expected online bot to be preferred, got %+v", response.Data[0])
	}
}

func TestGetAllOnlineBotEmptyWhenNoSource(t *testing.T) {
	resetBotSelectorState(t)

	req := httptest.NewRequest(http.MethodGet, "/bot/online", nil)
	recorder := httptest.NewRecorder()

	GetAllOnlineBot(recorder, req)

	var response struct {
		Code int                    `json:"code"`
		Data []checkpoint.BotStatus `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Code != 0 {
		t.Fatalf("expected success code, got %d, body=%s", response.Code, recorder.Body.String())
	}
	if len(response.Data) != 0 {
		t.Fatalf("expected empty list when no map and no db bot, got %d", len(response.Data))
	}
}
