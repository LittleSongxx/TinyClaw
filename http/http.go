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
		protected := withManagementAuth

		mux.HandleFunc("/user/token/add", protected(AddUserToken))
		mux.HandleFunc("/user/delete", protected(DeleteUser))

		mux.HandleFunc("/conf/update", protected(UpdateConf))
		mux.HandleFunc("/conf/get", protected(GetConf))
		mux.HandleFunc("/command/get", protected(GetCommand))
		mux.HandleFunc("/restart", protected(Restart))
		mux.HandleFunc("/stop", protected(Stop))
		mux.HandleFunc("/log", protected(Log))

		mux.HandleFunc("/mcp/get", protected(GetMCPConf))
		mux.HandleFunc("/mcp/inspect", protected(InspectMCPConf))
		mux.HandleFunc("/mcp/update", protected(UpdateMCPConf))
		mux.HandleFunc("/mcp/disable", protected(DisableMCPConf))
		mux.HandleFunc("/mcp/delete", protected(DeleteMCPConf))
		mux.HandleFunc("/mcp/sync", protected(SyncMCPConf))

		mux.HandleFunc("/user/list", protected(GetUsers))
		mux.HandleFunc("/user/quota/stats", protected(GetUserQuotaStats))
		mux.HandleFunc("/user/insert/record", protected(InsertUserRecords))
		mux.HandleFunc("/record/list", protected(GetRecords))
		mux.HandleFunc("/record/delete", protected(DeleteRecord))
		mux.HandleFunc("/run/list", protected(GetAgentRuns))
		mux.HandleFunc("/run/get", protected(GetAgentRun))
		mux.HandleFunc("/run/replay", protected(ReplayAgentRun))
		mux.HandleFunc("/run/delete", protected(DeleteAgentRun))
		mux.HandleFunc("/runs", protected(RunsHandler))
		mux.HandleFunc("GET /runs/{id}", protected(GetRunByPath))
		mux.HandleFunc("/tools/effective", protected(GetEffectiveTools))
		mux.HandleFunc("/skills/status", protected(GetSkillsStatus))
		mux.HandleFunc("/memory/status", protected(GetMemoryStatus))
		mux.HandleFunc("/knowledge/status", protected(GetKnowledgeStatus))
		mux.HandleFunc("/skills/list", protected(ListSkills))
		mux.HandleFunc("/skills/detail", protected(GetSkillDetail))
		mux.HandleFunc("/skills/reload", protected(ReloadSkills))
		mux.HandleFunc("/skills/validate", protected(ValidateSkills))

		if conf.FeatureConfInfo.KnowledgeEnabled() {
			mux.HandleFunc("/knowledge/search", protected(KnowledgeSearch))
			mux.HandleFunc("/knowledge/ingest", protected(KnowledgeIngest))
			mux.HandleFunc("/knowledge/files/list", protected(ListKnowledgeFiles))
			mux.HandleFunc("/knowledge/files/delete", protected(DeleteKnowledgeFile))
			mux.HandleFunc("/knowledge/files/create", protected(CreateKnowledgeFile))
			mux.HandleFunc("/knowledge/files/get", protected(GetKnowledgeFileContent))
			mux.HandleFunc("/knowledge/clear", protected(ClearKnowledgeData))
			mux.HandleFunc("/knowledge/collections/list", protected(ListKnowledgeCollections))
			mux.HandleFunc("/knowledge/collections/create", protected(CreateKnowledgeCollection))
			mux.HandleFunc("/knowledge/documents/list", protected(ListKnowledgeDocuments))
			mux.HandleFunc("/knowledge/documents/get", protected(GetKnowledgeDocument))
			mux.HandleFunc("/knowledge/documents/create", protected(CreateKnowledgeDocument))
			mux.HandleFunc("/knowledge/documents/delete", protected(DeleteKnowledgeDocument))
			mux.HandleFunc("/knowledge/jobs/list", protected(ListKnowledgeJobs))
			mux.HandleFunc("/knowledge/retrieval/debug", protected(DebugKnowledgeRetrieval))
			mux.HandleFunc("/knowledge/retrieval/runs/list", protected(ListKnowledgeRetrievalRuns))
			mux.HandleFunc("/knowledge/retrieval/runs/get", protected(GetKnowledgeRetrievalRun))
		}

		mux.HandleFunc("/pong", PongHandler)
		mux.HandleFunc("/dashboard", protected(DashboardHandler))
		mux.HandleFunc("/gateway/ws", GatewayWS)
		mux.HandleFunc("/gateway/nodes/ws", GatewayNodesWS)
		mux.HandleFunc("/gateway/nodes/list", protected(GetGatewayNodes))
		mux.HandleFunc("/gateway/sessions/list", protected(GetGatewaySessions))
		mux.HandleFunc("/gateway/approvals/list", protected(GetGatewayApprovals))
		mux.HandleFunc("/gateway/node/command", protected(ExecuteGatewayNodeCommand))
		mux.HandleFunc("/gateway/approvals/decide", protected(DecideGatewayApproval))
		mux.HandleFunc("/doctor/run", protected(DoctorRun))
		mux.HandleFunc("/security/audit", protected(SecurityAudit))
		mux.HandleFunc("/devices/bootstrap", protected(DeviceBootstrap))
		mux.HandleFunc("/devices/pending", protected(DevicePending))
		mux.HandleFunc("/devices/approve", protected(DeviceApprove))
		mux.HandleFunc("/devices/reject", protected(DeviceReject))
		mux.HandleFunc("/devices/revoke", protected(DeviceRevoke))
		mux.HandleFunc("/plugins/list", protected(PluginsList))
		mux.HandleFunc("/plugins/status", protected(PluginsStatus))
		mux.HandleFunc("/plugins/enable", protected(PluginsEnable))
		mux.HandleFunc("/plugins/disable", protected(PluginsDisable))
		mux.HandleFunc("/plugins/validate", protected(PluginsValidate))
		mux.HandleFunc("/flows/create", protected(FlowsCreate))
		mux.HandleFunc("/flows/update", protected(FlowsCreate))
		mux.HandleFunc("/flows/list", protected(FlowsList))
		mux.HandleFunc("/flows/get", protected(FlowsGet))
		mux.HandleFunc("/flows/validate", protected(FlowsValidate))
		mux.HandleFunc("/flows/run", protected(FlowsRun))
		mux.HandleFunc("/flow-runs/get", protected(FlowRunGet))
		mux.HandleFunc("/flow-runs/cancel", protected(FlowRunCancel))
		mux.HandleFunc("/flow-runs/retry-node", protected(FlowRunRetryNode))

		mux.HandleFunc("/communicate", Communicate)
		if conf.FeatureConfInfo.LegacyBotsEnabled() {
			mux.HandleFunc("/com/wechat", ComWechatComm)
			mux.HandleFunc("/wechat", WechatComm)
			mux.HandleFunc("/qq", QQBotComm)
			mux.HandleFunc("/onebot", OneBot)
		}

		if conf.FeatureConfInfo.CronEnabled() {
			mux.HandleFunc("/cron/create", protected(CreateCron))
			mux.HandleFunc("/cron/update", protected(UpdateCron))
			mux.HandleFunc("/cron/update_status", protected(UpdateCronStatus))
			mux.HandleFunc("/cron/delete", protected(DeleteCron))
			mux.HandleFunc("/cron/list", protected(GetCrons))
		}

		if conf.FeatureConfInfo.MediaEnabled() {
			mux.HandleFunc("/image", imageHandler)
		}

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
