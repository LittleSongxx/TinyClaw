package http

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/llm"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/utils"
)

func GetAgentRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	page := utils.ParseInt(r.FormValue("page"))
	pageSize := utils.ParseInt(r.FormValue("page_size"))
	if pageSize == 0 {
		pageSize = utils.ParseInt(r.FormValue("pageSize"))
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	mode := r.FormValue("mode")
	status := r.FormValue("status")
	userId := r.FormValue("user_id")
	if userId == "" {
		userId = r.FormValue("userId")
	}

	runs, err := db.GetAgentRunsByPage(page, pageSize, mode, status, userId)
	if err != nil {
		logger.ErrorCtx(ctx, "query agent runs fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	total, err := db.GetAgentRunsCount(mode, status, userId)
	if err != nil {
		logger.ErrorCtx(ctx, "query agent run count fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	utils.Success(ctx, w, r, map[string]interface{}{
		"list":      runs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"mode":      mode,
		"status":    status,
		"user_id":   userId,
	})
}

func GetAgentRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	id := utils.ParseInt(r.FormValue("id"))
	if id == 0 {
		utils.Failure(ctx, w, r, param.CodeParamError, "ID is required", nil)
		return
	}

	detail, err := db.GetAgentRunDetailByID(int64(id))
	if err != nil {
		logger.ErrorCtx(ctx, "query agent run detail fail", "err", err, "id", id)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	utils.Success(ctx, w, r, detail)
}

func ReplayAgentRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	id := utils.ParseInt(r.FormValue("id"))
	if id == 0 {
		utils.Failure(ctx, w, r, param.CodeParamError, "ID is required", nil)
		return
	}

	run, err := db.GetAgentRunByID(int64(id))
	if err != nil {
		logger.ErrorCtx(ctx, "query agent run fail", "err", err, "id", id)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	replayCtx, err := ensureReplayUserContext(ctx, run.UserId)
	if err != nil {
		logger.ErrorCtx(ctx, "prepare replay context fail", "err", err, "user_id", run.UserId)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	req := &llm.LLMTaskReq{
		Content:   run.Input,
		UserId:    run.UserId,
		ChatId:    run.ChatId,
		MsgId:     run.MsgId,
		PerMsgLen: llm.OneMsgLen,
		Cs:        &param.ContextState{UseRecord: true},
		Ctx:       replayCtx,
		ReplayOf:  run.ID,
	}

	var replayed *db.AgentRun
	switch run.Mode {
	case "mcp":
		req.SkillID = run.SkillID
		replayed, err = req.ExecuteMcpRun()
	case "skill":
		req.SkillID = run.SkillID
		replayed, err = req.ExecuteSkillRun()
	default:
		replayed, err = req.ExecuteTaskRun()
	}
	if err != nil {
		logger.ErrorCtx(ctx, "replay agent run fail", "err", err, "id", id)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}

	detail, err := db.GetAgentRunDetailByID(replayed.ID)
	if err != nil {
		logger.ErrorCtx(ctx, "query replayed run detail fail", "err", err, "id", replayed.ID)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	utils.Success(ctx, w, r, detail)
}

func DeleteAgentRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	id := utils.ParseInt(r.FormValue("id"))
	if id == 0 {
		id = utils.ParseInt(r.FormValue("run_id"))
	}
	if id == 0 {
		utils.Failure(ctx, w, r, param.CodeParamError, "ID is required", nil)
		return
	}

	err := db.DeleteAgentRunByID(int64(id))
	if err != nil {
		if err == sql.ErrNoRows {
			utils.Failure(ctx, w, r, param.CodeDBQueryFail, "run not found", err)
			return
		}
		logger.ErrorCtx(ctx, "delete agent run fail", "err", err, "id", id)
		utils.Failure(ctx, w, r, param.CodeDBWriteFail, param.MsgDBWriteFail, err)
		return
	}

	utils.Success(ctx, w, r, "success")
}

func ensureReplayUserContext(ctx context.Context, userID string) (context.Context, error) {
	userInfo, err := db.GetUserByID(userID)
	if err != nil {
		return ctx, err
	}
	if userInfo == nil || userInfo.ID == 0 {
		if _, err = db.InsertUser(userID, utils.GetDefaultLLMConfig()); err != nil {
			return ctx, err
		}
		userInfo, err = db.GetUserByID(userID)
		if err != nil {
			return ctx, err
		}
	}
	if userInfo == nil {
		return ctx, param.ErrDBQueryFail
	}
	if userInfo.LLMConfigRaw == nil {
		userInfo.LLMConfigRaw = new(param.LLMConfig)
	}
	return context.WithValue(ctx, "user_info", userInfo), nil
}
