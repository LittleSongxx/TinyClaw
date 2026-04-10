package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
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
	items, err := gateway.DefaultService().ListSessionMetaInWorkspace(ctx, 100)
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
	normalizeManagedNodeCommandRequest(r, &req)

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

func normalizeManagedNodeCommandRequest(r *http.Request, req *node.NodeCommandRequest) {
	if req == nil {
		return
	}

	if actingUserID := strings.TrimSpace(actingUserIDFromRequest(r)); actingUserID != "" {
		req.UserID = actingUserID
	}
	if principal, ok := authz.PrincipalFromContext(r.Context()); ok {
		req.WorkspaceID = principal.WorkspaceID
		req.ActorID = principal.ActorID
		req.ActorRole = string(principal.Role)
		req.ActorScopes = append([]string(nil), principal.Scopes...)
	}

	req.RequireApproval = true
}
