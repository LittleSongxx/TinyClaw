package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/gorilla/websocket"
)

func main() {
	opts, explicit, err := parseCLIOptions()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if opts.Configure {
		if err := runConfigureMode(opts.ConfigPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := resolveProcessConfig(ctx, opts, explicit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger.InitLoggerWithFile(filepath.Join(cfg.LogDir, "tiny_claw.log"))

	instances, err := buildNodeInstances(ctx, cfg)
	if err != nil {
		logger.Error("build node instances failed", "err", err)
		os.Exit(1)
	}

	logger.Info("tinyclaw-node starting",
		"gateway_ws", cfg.GatewayWS,
		"instances", len(instances),
		"windows_enabled", cfg.EnableWindowsNode,
	)

	var waitGroup sync.WaitGroup
	for _, instance := range instances {
		waitGroup.Add(1)
		go func(current nodeInstance) {
			defer waitGroup.Done()
			runNodeLoop(ctx, cfg.GatewayWS, cfg.NodeToken, current)
		}(instance)
	}

	<-ctx.Done()
	logger.Info("tinyclaw-node shutting down")
	waitGroup.Wait()
}

func hostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func writeJSON(mu *sync.Mutex, conn *websocket.Conn, payload interface{}) error {
	mu.Lock()
	defer mu.Unlock()
	return conn.WriteJSON(payload)
}
