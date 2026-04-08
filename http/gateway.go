package http

import (
	"net/http"
	"time"

	"github.com/LittleSongxx/TinyClaw/gateway"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/utils"
)

func GatewayWS(w http.ResponseWriter, r *http.Request) {
	gateway.DefaultService().HandleControlWS(w, r)
}

func GatewayNodesWS(w http.ResponseWriter, r *http.Request) {
	gateway.DefaultService().HandleNodeWS(w, r)
}

func GetGatewayNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	utils.Success(ctx, w, r, gateway.DefaultService().ListNodes(ctx))
}

func GetGatewaySessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	items, err := gateway.DefaultService().ListSessionMeta(100)
	if err != nil {
		logger.ErrorCtx(ctx, "list gateway sessions fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}
	utils.Success(ctx, w, r, items)
}

func GetGatewayApprovals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	utils.Success(ctx, w, r, gateway.DefaultService().ListApprovals(ctx))
}

func ExecuteGatewayNodeCommand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req node.NodeCommandRequest
	if err := utils.HandleJsonBody(r, &req); err != nil {
		logger.ErrorCtx(ctx, "parse node command body fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	result, err := gateway.DefaultService().ExecuteNodeCommand(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "execute gateway node command fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}

	utils.Success(ctx, w, r, result)
}

func DecideGatewayApproval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var decision node.ApprovalDecision
	if err := utils.HandleJsonBody(r, &decision); err != nil {
		logger.ErrorCtx(ctx, "parse approval decision body fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}
	if decision.CommandID == "" {
		utils.Failure(ctx, w, r, param.CodeParamError, "command_id is required", nil)
		return
	}
	if decision.CreatedAt == 0 {
		decision.CreatedAt = time.Now().Unix()
	}

	result, err := gateway.DefaultService().DecideApproval(ctx, decision)
	if err != nil {
		logger.ErrorCtx(ctx, "decide gateway approval fail", "err", err, "command_id", decision.CommandID)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}
