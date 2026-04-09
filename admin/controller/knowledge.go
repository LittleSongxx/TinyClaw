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

func ListKnowledgeFiles(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/knowledge/files/list", nil)
}

func GetKnowledgeFile(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/knowledge/files/get", nil)
}

func DeleteKnowledgeFile(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/knowledge/files/delete", nil)
}

func CreateKnowledgeFile(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/knowledge/files/create", r.Body)
}

func ListKnowledgeCollections(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/knowledge/collections/list", nil)
}

func CreateKnowledgeCollection(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/knowledge/collections/create", r.Body)
}

func ListKnowledgeDocuments(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/knowledge/documents/list", nil)
}

func GetKnowledgeDocument(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/knowledge/documents/get", nil)
}

func CreateKnowledgeDocument(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/knowledge/documents/create", r.Body)
}

func DeleteKnowledgeDocument(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/knowledge/documents/delete", nil)
}

func ListKnowledgeJobs(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/knowledge/jobs/list", nil)
}

func DebugKnowledgeRetrieval(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodPost, "/knowledge/retrieval/debug", r.Body)
}

func ListKnowledgeRetrievalRuns(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/knowledge/retrieval/runs/list", nil)
}

func GetKnowledgeRetrievalRun(w http.ResponseWriter, r *http.Request) {
	proxyBotRequest(w, r, http.MethodGet, "/knowledge/retrieval/runs/get", nil)
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
