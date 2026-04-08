package http

import (
	"net/http"

	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/skill"
	"github.com/LittleSongxx/TinyClaw/utils"
)

func ListSkills(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	catalog, err := skill.LoadDefaultCatalog()
	if err != nil {
		logger.ErrorCtx(ctx, "load skills fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}

	utils.Success(ctx, w, r, map[string]interface{}{
		"list":     catalog.List(),
		"warnings": catalog.Warnings,
	})
}

func GetSkillDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	id := r.FormValue("id")
	if id == "" {
		utils.Failure(ctx, w, r, param.CodeParamError, "skill id is required", nil)
		return
	}

	catalog, err := skill.LoadDefaultCatalog()
	if err != nil {
		logger.ErrorCtx(ctx, "load skills fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}

	item, ok := catalog.Get(id)
	if !ok {
		utils.Failure(ctx, w, r, param.CodeParamError, "skill not found", nil)
		return
	}

	utils.Success(ctx, w, r, item)
}

func ReloadSkills(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	catalog, err := skill.LoadDefaultCatalog()
	if err != nil {
		logger.ErrorCtx(ctx, "reload skills fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}

	utils.Success(ctx, w, r, map[string]interface{}{
		"reloaded": true,
		"count":    len(catalog.List()),
		"warnings": catalog.Warnings,
	})
}

func ValidateSkills(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	report, err := skill.ValidateDefaultCatalog()
	if err != nil {
		logger.ErrorCtx(ctx, "validate skills fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}

	utils.Success(ctx, w, r, report)
}
