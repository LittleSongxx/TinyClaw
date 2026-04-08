package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/metrics"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	FilterPath = map[string]bool{
		"/pong": true,
	}
)

type HTTPServer struct {
	Addr string
}

func InitHTTP() {
	initImg()
	pprofServer := NewHTTPServer(fmt.Sprintf("%s", conf.BaseConfInfo.HTTPHost))
	pprofServer.Start()
}

// NewHTTPServer create http server, listen 36060 port.
func NewHTTPServer(addr string) *HTTPServer {
	if addr == "" {
		addr = ":36060"
	}
	return &HTTPServer{
		Addr: addr,
	}
}

// Start pprof server
func (p *HTTPServer) Start() {
	go func() {
		logger.Info("Starting pprof server on", "addr", p.Addr)
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())

		mux.HandleFunc("/user/token/add", AddUserToken)
		mux.HandleFunc("/user/delete", DeleteUser)

		mux.HandleFunc("/conf/update", UpdateConf)
		mux.HandleFunc("/conf/get", GetConf)
		mux.HandleFunc("/command/get", GetCommand)
		mux.HandleFunc("/restart", Restart)
		mux.HandleFunc("/stop", Stop)
		mux.HandleFunc("/log", Log)

		mux.HandleFunc("/mcp/get", GetMCPConf)
		mux.HandleFunc("/mcp/inspect", InspectMCPConf)
		mux.HandleFunc("/mcp/update", UpdateMCPConf)
		mux.HandleFunc("/mcp/disable", DisableMCPConf)
		mux.HandleFunc("/mcp/delete", DeleteMCPConf)
		mux.HandleFunc("/mcp/sync", SyncMCPConf)

		mux.HandleFunc("/user/list", GetUsers)
		mux.HandleFunc("/user/quota/stats", GetUserQuotaStats)
		mux.HandleFunc("/user/insert/record", InsertUserRecords)
		mux.HandleFunc("/record/list", GetRecords)
		mux.HandleFunc("/record/delete", DeleteRecord)
		mux.HandleFunc("/run/list", GetAgentRuns)
		mux.HandleFunc("/run/get", GetAgentRun)
		mux.HandleFunc("/run/replay", ReplayAgentRun)
		mux.HandleFunc("/run/delete", DeleteAgentRun)
		mux.HandleFunc("/skills/list", ListSkills)
		mux.HandleFunc("/skills/detail", GetSkillDetail)
		mux.HandleFunc("/skills/reload", ReloadSkills)
		mux.HandleFunc("/skills/validate", ValidateSkills)

		mux.HandleFunc("/rag/list", GetRagFile)
		mux.HandleFunc("/rag/delete", DeleteRagFile)
		mux.HandleFunc("/rag/create", CreateRagFile)
		mux.HandleFunc("/rag/get", GetRagFileContent)
		mux.HandleFunc("/rag/clear", ClearAllVectorData)
		mux.HandleFunc("/rag/collections/list", ListRagCollections)
		mux.HandleFunc("/rag/collections/create", CreateRagCollection)
		mux.HandleFunc("/rag/documents/list", ListRagDocuments)
		mux.HandleFunc("/rag/documents/get", GetRagDocument)
		mux.HandleFunc("/rag/documents/create", CreateRagDocument)
		mux.HandleFunc("/rag/documents/delete", DeleteRagDocument)
		mux.HandleFunc("/rag/jobs/list", ListRagJobs)
		mux.HandleFunc("/rag/retrieval/debug", DebugRagRetrieval)
		mux.HandleFunc("/rag/retrieval/runs/list", ListRagRetrievalRuns)
		mux.HandleFunc("/rag/retrieval/runs/get", GetRagRetrievalRun)

		mux.HandleFunc("/pong", PongHandler)
		mux.HandleFunc("/dashboard", DashboardHandler)
		mux.HandleFunc("/gateway/ws", GatewayWS)
		mux.HandleFunc("/gateway/nodes/ws", GatewayNodesWS)
		mux.HandleFunc("/gateway/nodes/list", GetGatewayNodes)
		mux.HandleFunc("/gateway/sessions/list", GetGatewaySessions)
		mux.HandleFunc("/gateway/approvals/list", GetGatewayApprovals)
		mux.HandleFunc("/gateway/node/command", ExecuteGatewayNodeCommand)
		mux.HandleFunc("/gateway/approvals/decide", DecideGatewayApproval)

		mux.HandleFunc("/communicate", Communicate)
		mux.HandleFunc("/com/wechat", ComWechatComm)
		mux.HandleFunc("/wechat", WechatComm)
		mux.HandleFunc("/qq", QQBotComm)
		mux.HandleFunc("/onebot", OneBot)

		mux.HandleFunc("/cron/create", CreateCron)
		mux.HandleFunc("/cron/update", UpdateCron)
		mux.HandleFunc("/cron/update_status", UpdateCronStatus)
		mux.HandleFunc("/cron/delete", DeleteCron)
		mux.HandleFunc("/cron/list", GetCrons)

		mux.HandleFunc("/image", imageHandler)

		wrappedMux := WithRequestContext(mux)

		var err error
		if conf.BaseConfInfo.CrtFile == "" || conf.BaseConfInfo.KeyFile == "" {
			err = http.ListenAndServe(p.Addr, wrappedMux)
		} else {
			err = runTLSServer(wrappedMux)
		}
		if err != nil {
			logger.Fatal("pprof server failed", "err", err)
		}
	}()
}

func runTLSServer(wrappedMux http.Handler) error {
	caCert, err := os.ReadFile(conf.BaseConfInfo.CaFile)
	if err != nil {
		return err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(conf.BaseConfInfo.CrtFile, conf.BaseConfInfo.KeyFile)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	server := &http.Server{
		Addr:      fmt.Sprintf("%s", conf.BaseConfInfo.HTTPHost),
		TLSConfig: tlsConfig,
		Handler:   wrappedMux,
	}

	err = server.ListenAndServeTLS("", "")
	if err != nil {
		return err
	}

	return nil
}

func WithRequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		isSSE := r.Header.Get("Accept") == "text/event-stream"
		isWebSocket := strings.EqualFold(r.Header.Get("Upgrade"), "websocket")

		var cancel context.CancelFunc
		if !isSSE && !isWebSocket {
			ctx, cancel = context.WithTimeout(ctx, 15*time.Minute)
			defer cancel()
		}

		logID := r.Header.Get("LogId")
		if logID == "" {
			logID = uuid.New().String()
		}
		ctx = context.WithValue(ctx, "log_id", logID)

		if conf.BaseConfInfo.BotName != "" {
			ctx = context.WithValue(ctx, "bot_name", conf.BaseConfInfo.BotName)
		}

		ctx = context.WithValue(ctx, "start_time", time.Now())

		r = r.WithContext(ctx)

		if !FilterPath[r.URL.Path] {
			logger.InfoCtx(ctx, "request start", "path", r.URL.Path)
		}

		next.ServeHTTP(w, r)

		metrics.HTTPRequestCount.WithLabelValues(r.URL.Path).Inc()
	})
}
