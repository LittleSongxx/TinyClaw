package db

import (
	"testing"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
	_ "github.com/mattn/go-sqlite3"
)

func TestInsertAndGetUser(t *testing.T) {
	conf.BaseConfInfo.TokenPerUser = 100

	userId := "123456789-" + time.Now().Format("150405.000000000")
	mode := `{"txt_type":"gemini","txt_model":"gemini-2.0-flash","img_type":"gemini","img_model":"gemini-2.0-flash-preview-image-generation","video_type":"gemini","video_model":"veo-2.0-generate-001"}`

	// 插入用户
	id, err := InsertUser(userId, mode)
	if err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero ID")
	}

	// 获取用户
	user, err := GetUserByID(userId)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if user == nil {
		t.Fatalf("user not found")
	}
	if user.UserId != userId || user.LLMConfig != mode || user.AvailToken != 100 {
		t.Errorf("unexpected user data: %+v", user)
	}

	users, err := GetUsers()
	if err != nil {
		t.Fatalf("GetUsers failed: %v", err)
	}
	if len(users) == 0 {
		t.Fatalf("user not found")
	}

	err = UpdateUserLLMConfig(user.UserId, `{"txt_type":"gemini","txt_model":"gemini-2.0-flash","img_type":"gemini","img_model":"gemini-2.0-flash-preview-image-generation","video_type":"gemini","video_model":"veo-2.0-generate-001"}`)
	if err != nil {
		t.Fatalf("UpdateUserMode failed: %v", err)
	}

	err = AddAvailToken(user.UserId, 1000)
	if err != nil {
		t.Fatalf("UpdateUserUpdateTime failed: %v", err)
	}

	err = AddToken(user.UserId, 1000)
	if err != nil {
		t.Fatalf("AddToken failed: %v", err)
	}

	user, err = GetUserByID(userId)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if user.UserId != userId || user.LLMConfig != `{"txt_type":"gemini","txt_model":"gemini-2.0-flash","img_type":"gemini","img_model":"gemini-2.0-flash-preview-image-generation","video_type":"gemini","video_model":"veo-2.0-generate-001"}` || user.Token != 1000 || user.AvailToken != 1100 {
		t.Errorf("unexpected user data: %+v", user)
	}

}

func TestPrivilegedUserHasUnlimitedQuota(t *testing.T) {
	userID := "privileged-user"
	original := conf.BaseConfInfo.PrivilegedUserIds
	conf.BaseConfInfo.PrivilegedUserIds = map[string]bool{userID: true}
	t.Cleanup(func() {
		conf.BaseConfInfo.PrivilegedUserIds = original
	})

	if _, err := InsertUser(userID, "{}"); err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}

	user, err := GetUserByID(userID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if user == nil {
		t.Fatalf("user not found")
	}
	if !user.Unlimited {
		t.Fatalf("expected privileged user to be marked unlimited")
	}
	if user.AvailToken != -1 {
		t.Fatalf("expected unlimited quota sentinel, got %d", user.AvailToken)
	}

	stats, err := GetUserQuotaStats(1, 10, userID, "low_remaining")
	if err != nil {
		t.Fatalf("GetUserQuotaStats failed: %v", err)
	}
	if stats.Summary.UnlimitedUsers != 1 {
		t.Fatalf("expected 1 unlimited user, got %d", stats.Summary.UnlimitedUsers)
	}
	if len(stats.List) != 1 || !stats.List[0].Unlimited {
		t.Fatalf("expected unlimited metric in list: %+v", stats.List)
	}
}
