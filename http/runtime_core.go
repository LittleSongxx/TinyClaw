package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/recall"
	"github.com/LittleSongxx/TinyClaw/runtimecore"
	"github.com/LittleSongxx/TinyClaw/utils"
)

type RuntimeRunRequest struct {
	Mode      string `json:"mode"`
	Input     string `json:"input"`
	UserID    string `json:"user_id"`
	ChatID    string `json:"chat_id"`
	MsgID     string `json:"msg_id"`
	SkillID   string `json:"skill_id"`
	ReplayOf  int64  `json:"replay_of"`
	PerMsgLen int    `json:"per_msg_len"`
	UseRecall *bool  `json:"use_recall,omitempty"`
}

type KnowledgeSearchRequest struct {
	Query string `json:"query"`
}

func RunsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		CreateRun(w, r)
	case http.MethodGet:
		if id := r.URL.Query().Get("id"); id != "" {
			GetRunByPath(w, r)
			return
		}
		GetAgentRuns(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func CreateRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := new(RuntimeRunRequest)
	if err := utils.HandleJsonBody(r, req); err != nil {
		logger.ErrorCtx(ctx, "parse runtime run request fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	if strings.TrimSpace(req.Input) == "" || strings.TrimSpace(req.UserID) == "" {
		utils.Failure(ctx, w, r, param.CodeParamError, "input and user_id are required", nil)
		return
	}

	result, err := runtimecore.DefaultService().Run(runtimecore.RunRequest{
		Ctx:       ctx,
		Mode:      runtimecore.Mode(strings.TrimSpace(req.Mode)),
		Input:     req.Input,
		UserID:    req.UserID,
		ChatID:    req.ChatID,
		MsgID:     req.MsgID,
		ReplayOf:  req.ReplayOf,
		SkillID:   req.SkillID,
		PerMsgLen: req.PerMsgLen,
		UseRecall: req.UseRecall,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "runtime run fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}

	if result.Run != nil && result.Run.ID != 0 {
		detail, detailErr := db.GetAgentRunDetailByID(result.Run.ID)
		if detailErr == nil {
			utils.Success(ctx, w, r, detail)
			return
		}
	}

	utils.Success(ctx, w, r, result)
}

func GetRunByPath(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawID := r.PathValue("id")
	if rawID == "" {
		rawID = r.URL.Query().Get("id")
	}
	id, _ := strconv.ParseInt(rawID, 10, 64)
	if id == 0 {
		utils.Failure(ctx, w, r, param.CodeParamError, "ID is required", nil)
		return
	}

	detail, err := db.GetAgentRunDetailByID(id)
	if err != nil {
		logger.ErrorCtx(ctx, "query agent run detail fail", "err", err, "id", id)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}
	utils.Success(ctx, w, r, detail)
}

func GetEffectiveTools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result, err := runtimecore.DefaultService().EffectiveTools(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "list effective tools fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func GetSkillsStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result, err := runtimecore.DefaultService().SkillsStatus(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "list skills status fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func GetMemoryStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	utils.Success(ctx, w, r, recall.DefaultService().MemoryStatus(ctx))
}

func GetKnowledgeStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	utils.Success(ctx, w, r, runtimecore.DefaultService().KnowledgeStatus(ctx))
}

func KnowledgeSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := new(KnowledgeSearchRequest)
	if err := utils.HandleJsonBody(r, req); err != nil {
		logger.ErrorCtx(ctx, "parse knowledge search request fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		utils.Failure(ctx, w, r, param.CodeParamError, "query is required", nil)
		return
	}

	hits, err := runtimecore.DefaultService().SearchKnowledge(ctx, req.Query)
	if err != nil {
		logger.ErrorCtx(ctx, "knowledge search fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, map[string]interface{}{
		"query": req.Query,
		"hits":  hits,
	})
}

func KnowledgeIngest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	CreateKnowledgeDocument(w, r.WithContext(ctx))
}
