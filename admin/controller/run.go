package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	adminUtils "github.com/LittleSongxx/TinyClaw/admin/utils"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/utils"
)

func ListRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	botInfo, err := getBot(r)
	if err != nil {
		logger.ErrorCtx(ctx, "get bot conf error", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	if err = r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	targetURL := strings.TrimSuffix(botInfo.Address, "/") + fmt.Sprintf(
		"/run/list?page=%s&page_size=%s&mode=%s&status=%s&user_id=%s",
		r.FormValue("page"), r.FormValue("pageSize"), r.FormValue("mode"), r.FormValue("status"), r.FormValue("userId"),
	)

	resp, err := adminUtils.GetCrtClient(botInfo).Do(GetRequest(ctx, http.MethodGet, targetURL, bytes.NewBuffer(nil)))
	if err != nil {
		logger.ErrorCtx(ctx, "request run list error", "err", err)
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

func GetRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	botInfo, err := getBot(r)
	if err != nil {
		logger.ErrorCtx(ctx, "get bot conf error", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	if err = r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	runID := r.FormValue("run_id")
	if runID == "" {
		runID = r.FormValue("id")
	}

	targetURL := strings.TrimSuffix(botInfo.Address, "/") + "/run/get?id=" + runID
	resp, err := adminUtils.GetCrtClient(botInfo).Do(GetRequest(ctx, http.MethodGet, targetURL, bytes.NewBuffer(nil)))
	if err != nil {
		logger.ErrorCtx(ctx, "request run detail error", "err", err)
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

func ReplayRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	botInfo, err := getBot(r)
	if err != nil {
		logger.ErrorCtx(ctx, "get bot conf error", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	if err = r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	runID := r.FormValue("run_id")
	if runID == "" {
		runID = r.FormValue("id")
	}

	targetURL := strings.TrimSuffix(botInfo.Address, "/") + "/run/replay?id=" + runID
	resp, err := adminUtils.GetCrtClient(botInfo).Do(GetRequest(ctx, http.MethodPost, targetURL, bytes.NewBuffer(nil)))
	if err != nil {
		logger.ErrorCtx(ctx, "request replay run error", "err", err)
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
