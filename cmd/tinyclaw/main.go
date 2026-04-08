//go:build !libtokenizers

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/http"
	"github.com/LittleSongxx/TinyClaw/i18n"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/metrics"
	"github.com/LittleSongxx/TinyClaw/rag"
	"github.com/LittleSongxx/TinyClaw/register"
	"github.com/LittleSongxx/TinyClaw/robot"
	"github.com/LittleSongxx/TinyClaw/skill"
)

func main() {
	logger.InitLogger()
	conf.InitConf()
	i18n.InitI18n()
	db.InitTable()
	conf.InitTools()
	skill.LogDefaultCatalog(context.Background())
	rag.InitRag()
	http.InitHTTP()
	metrics.RegisterMetrics()
	robot.StartRobot()
	register.InitRegister()
	robot.InitCron()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
