//go:build !libtokenizers

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/doctor"
	"github.com/LittleSongxx/TinyClaw/gateway"
	"github.com/LittleSongxx/TinyClaw/http"
	"github.com/LittleSongxx/TinyClaw/i18n"
	"github.com/LittleSongxx/TinyClaw/knowledge"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/metrics"
	"github.com/LittleSongxx/TinyClaw/register"
	"github.com/LittleSongxx/TinyClaw/robot"
	"github.com/LittleSongxx/TinyClaw/skill"
)

func main() {
	if handled := maybeRunDiagnosticCLI(); handled {
		return
	}
	logger.InitLogger()
	conf.InitConf()
	conf.InitRuntimeConf()
	if conf.RuntimeConfInfo.Nodes.LegacyNodeTokenPresent {
		logger.Fatal("NODE_PAIRING_TOKEN is no longer supported; remove it and pair devices with the Device Pairing flow")
	}
	i18n.InitI18n()
	db.InitTable()
	conf.InitTools()
	skill.LogDefaultCatalog(context.Background())
	knowledge.Init()
	gateway.Init()
	http.InitHTTP()
	metrics.RegisterMetrics()
	robot.StartRobot()
	register.InitRegister()
	if conf.FeatureConfInfo.CronEnabled() {
		robot.InitCron()
	} else {
		logger.Info("cron module disabled")
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func maybeRunDiagnosticCLI() bool {
	if len(os.Args) < 2 {
		return false
	}
	kind := ""
	args := []string{}
	switch os.Args[1] {
	case "doctor":
		kind = "doctor"
		args = os.Args[2:]
	case "security":
		if len(os.Args) >= 3 && os.Args[2] == "audit" {
			kind = "security_audit"
			args = os.Args[3:]
		}
	default:
		return false
	}
	flags := flag.NewFlagSet(kind, flag.ContinueOnError)
	workspaceID := flags.String("workspace", "default", "workspace id")
	asJSON := flags.Bool("json", false, "print JSON report")
	fix := flags.Bool("fix", false, "apply safe fixes only")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	logger.InitLogger()
	os.Args = []string{os.Args[0]}
	conf.InitConf()
	conf.InitRuntimeConf()
	db.InitTable()
	opts := doctor.Options{WorkspaceID: *workspaceID, Fix: *fix}
	var report doctor.Report
	if kind == "doctor" {
		report = doctor.Run(context.Background(), opts)
	} else {
		report = doctor.SecurityAudit(context.Background(), opts)
	}
	if *asJSON {
		body, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(body))
	} else {
		fmt.Printf("%s workspace=%s ok=%v\n", report.Kind, report.WorkspaceID, report.OK)
		for _, finding := range report.Findings {
			fmt.Printf("[%s] %s: %s\n", finding.Severity, finding.ID, finding.Message)
		}
	}
	if !report.OK {
		os.Exit(1)
	}
	return true
}
