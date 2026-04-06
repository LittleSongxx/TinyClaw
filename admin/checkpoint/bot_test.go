package checkpoint

import (
	"os"
	"testing"

	"github.com/LittleSongxx/TinyClaw/admin/conf"
	"github.com/LittleSongxx/TinyClaw/admin/db"
)

func TestMain(m *testing.M) {
	setup()

	code := m.Run()

	os.Exit(code)
}

func setup() {
	conf.InitConfig()
	db.InitTable()
}

func TestScheduleBotChecks(t *testing.T) {
	err := db.CreateBot("http://127.0.0.1:19019", "testbot", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	ScheduleBotChecks()

	BotMap.Range(func(key, value interface{}) bool {
		v, ok := value.(*BotStatus)
		if !ok {
			t.Fatal("value is not *BotStatus")
		}
		if v.Name == "testbot" && v.Status != "offline" {
			t.Fatal("bot status is not offline")
		}
		return true
	})
}
