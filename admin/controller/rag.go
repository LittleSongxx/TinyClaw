package controller

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	adminUtils "github.com/LittleSongxx/TinyClaw/admin/utils"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/utils"
)

func ListRagFiles(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/rag/list", nil)
}

func GetRagFile(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/rag/get", nil)
}

func DeleteRagFile(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/rag/delete", nil)
}

func CreateRagFile(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/rag/create", r.Body)
}

func ListRagCollectionsV2(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/rag/collections/list", nil)
}

func CreateRagCollectionV2(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/rag/collections/create", r.Body)
}

func ListRagDocuments(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/rag/documents/list", nil)
}

func GetRagDocument(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/rag/documents/get", nil)
}

func CreateRagDocument(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/rag/documents/create", r.Body)
}

func DeleteRagDocument(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/rag/documents/delete", nil)
}

func ListRagJobs(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/rag/jobs/list", nil)
}

func DebugRagRetrieval(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/rag/retrieval/debug", r.Body)
}

func ListRagRetrievalRuns(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/rag/retrieval/runs/list", nil)
}

func GetRagRetrievalRun(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/rag/retrieval/runs/get", nil)
}

func proxyBotRequest(w http.ResponseWriter, r *http.Request, method, suffix string, body io.Reader) {
	ctx := r.Context()
	botInfo, err := getBot(r)
	if err != nil {
		logger.ErrorCtx(ctx, "get bot conf error", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	targetURL := strings.TrimSuffix(botInfo.Address, "/") + suffix
	if rawQuery := forwardBotQuery(r); rawQuery != "" {
		targetURL += "?" + rawQuery
	}

	if body == nil {
		body = bytes.NewBuffer(nil)
	}

	resp, err := adminUtils.GetCrtClient(botInfo).Do(GetRequest(ctx, method, targetURL, body))
	if err != nil {
		logger.ErrorCtx(ctx, "proxy bot request error", "err", err, "url", targetURL)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	defer resp.Body.Close()

	if _, err = io.Copy(w, resp.Body); err != nil {
		logger.ErrorCtx(ctx, "copy response body error", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
}

func forwardBotQuery(r *http.Request) string {
	values := r.URL.Query()
	values.Del("id")
	return values.Encode()
}
