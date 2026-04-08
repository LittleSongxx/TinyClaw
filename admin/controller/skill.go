package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	adminUtils "github.com/LittleSongxx/TinyClaw/admin/utils"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/utils"
)

func ListSkills(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	botInfo, err := getBot(r)
	if err != nil {
		logger.ErrorCtx(ctx, "get bot conf error", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	targetURL := strings.TrimSuffix(botInfo.Address, "/") + "/skills/list"
	resp, err := adminUtils.GetCrtClient(botInfo).Do(GetRequest(ctx, http.MethodGet, targetURL, bytes.NewBuffer(nil)))
	if err != nil {
		logger.ErrorCtx(ctx, "request skill list error", "err", err)
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

func GetSkillDetail(w http.ResponseWriter, r *http.Request) {
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

	skillID := strings.TrimSpace(r.FormValue("skill_id"))
	if skillID == "" {
		utils.Failure(ctx, w, r, param.CodeParamError, "skill_id is required", nil)
		return
	}

	targetURL := strings.TrimSuffix(botInfo.Address, "/") + fmt.Sprintf("/skills/detail?id=%s", url.QueryEscape(skillID))
	resp, err := adminUtils.GetCrtClient(botInfo).Do(GetRequest(ctx, http.MethodGet, targetURL, bytes.NewBuffer(nil)))
	if err != nil {
		logger.ErrorCtx(ctx, "request skill detail error", "err", err)
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

func ReloadSkills(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	botInfo, err := getBot(r)
	if err != nil {
		logger.ErrorCtx(ctx, "get bot conf error", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	targetURL := strings.TrimSuffix(botInfo.Address, "/") + "/skills/reload"
	resp, err := adminUtils.GetCrtClient(botInfo).Do(GetRequest(ctx, http.MethodPost, targetURL, bytes.NewBuffer(nil)))
	if err != nil {
		logger.ErrorCtx(ctx, "request skill reload error", "err", err)
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

func ValidateSkills(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	botInfo, err := getBot(r)
	if err != nil {
		logger.ErrorCtx(ctx, "get bot conf error", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	targetURL := strings.TrimSuffix(botInfo.Address, "/") + "/skills/validate"
	resp, err := adminUtils.GetCrtClient(botInfo).Do(GetRequest(ctx, http.MethodGet, targetURL, bytes.NewBuffer(nil)))
	if err != nil {
		logger.ErrorCtx(ctx, "request skill validation error", "err", err)
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
