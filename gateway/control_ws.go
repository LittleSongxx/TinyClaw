package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/LittleSongxx/TinyClaw/plugins"
	"github.com/LittleSongxx/TinyClaw/taskflow"
)

func (s *Service) HandleControlWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var connect ConnectFrame
	if err := conn.ReadJSON(&connect); err != nil {
		return
	}
	principal, ok := s.authorizeControlConnect(connect)
	if connect.Type != FrameTypeConnect || connect.Role != "control" || !ok {
		return
	}
	ctx := authz.WithPrincipal(r.Context(), principal)

	readyFrame, err := NewEventFrame("gateway.ready", map[string]interface{}{
		"nodes": s.ListNodes(ctx),
	})
	if err == nil {
		_ = conn.WriteJSON(readyFrame)
	}

	for {
		var request RequestFrame
		if err := conn.ReadJSON(&request); err != nil {
			return
		}

		var response ResponseFrame
		method := request.MethodName()
		if err := request.Validate(); err != nil {
			response, err = NewResponseFrame(request.ID, false, nil, err.Error())
			if err == nil {
				_ = conn.WriteJSON(response)
			}
			continue
		}
		idempotencyKey := s.idempotencyKey(principal, method, request.IdempotencyKey)
		if cached, ok := s.idempotency.Get(idempotencyKey, time.Now()); ok {
			_ = conn.WriteJSON(cached)
			continue
		}

		switch method {
		case "nodes.list":
			response, err = NewResponseFrame(request.ID, true, s.ListNodes(ctx), "")
		case "sessions.list":
			items, listErr := s.ListSessionMetaInWorkspace(ctx, 100)
			if listErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, listErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, items, "")
			}
		case "node.exec", "node.command":
			var execReq node.NodeCommandRequest
			if body := request.RawParams(); len(body) > 0 {
				if decodeErr := json.Unmarshal(body, &execReq); decodeErr != nil {
					response, err = NewResponseFrame(request.ID, false, nil, decodeErr.Error())
					break
				}
			}
			normalizeControlExecRequest(ctx, &execReq)
			result, execErr := s.ExecuteNodeCommand(ctx, execReq)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, result, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "devices.bootstrap":
			result, execErr := s.CreateDeviceBootstrap(ctx)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "devices.pending", "devices.list":
			result, execErr := s.ListPendingDevices(ctx)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "devices.approve":
			var body struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(request.RawParams(), &body)
			result, execErr := s.ApproveDevice(ctx, body.RequestID)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "devices.reject":
			var body struct {
				RequestID string `json:"request_id"`
				Reason    string `json:"reason"`
			}
			_ = json.Unmarshal(request.RawParams(), &body)
			execErr := s.RejectDevice(ctx, body.RequestID, body.Reason)
			response, err = NewResponseFrame(request.ID, execErr == nil, map[string]bool{"ok": execErr == nil}, controlErrorText(execErr))
		case "devices.revoke":
			var body struct {
				DeviceID string `json:"device_id"`
			}
			_ = json.Unmarshal(request.RawParams(), &body)
			execErr := s.RevokeDevice(ctx, body.DeviceID)
			response, err = NewResponseFrame(request.ID, execErr == nil, map[string]bool{"ok": execErr == nil}, controlErrorText(execErr))
		case "plugins.list":
			result, execErr := s.PluginRegistry().List(ctx)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "plugins.status":
			var body struct {
				PluginID string `json:"plugin_id"`
			}
			_ = json.Unmarshal(request.RawParams(), &body)
			result, execErr := s.PluginRegistry().Status(ctx, body.PluginID)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "plugins.enable", "plugins.disable":
			var body struct {
				PluginID string `json:"plugin_id"`
				Config   string `json:"config"`
			}
			_ = json.Unmarshal(request.RawParams(), &body)
			execErr := s.PluginRegistry().SetEnabled(ctx, body.PluginID, method == "plugins.enable", body.Config)
			response, err = NewResponseFrame(request.ID, execErr == nil, map[string]bool{"ok": execErr == nil}, controlErrorText(execErr))
		case "plugins.reload":
			s.plugins = plugins.NewDefaultRegistry()
			result, execErr := s.PluginRegistry().List(ctx)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "plugins.validate":
			var manifest plugins.Manifest
			_ = json.Unmarshal(request.RawParams(), &manifest)
			execErr := s.PluginRegistry().Validate(ctx, manifest)
			response, err = NewResponseFrame(request.ID, execErr == nil, map[string]bool{"ok": execErr == nil}, controlErrorText(execErr))
		case "flows.create", "flows.update":
			var body struct {
				FlowID      string        `json:"flow_id"`
				Name        string        `json:"name"`
				Description string        `json:"description"`
				Spec        taskflow.Spec `json:"spec"`
			}
			_ = json.Unmarshal(request.RawParams(), &body)
			result, execErr := taskflow.CreateOrUpdate(ctx, body.FlowID, body.Name, body.Description, body.Spec)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "flows.list":
			result, execErr := db.ListTaskFlows(ctx, principal.WorkspaceID)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "flows.get":
			var body struct {
				FlowID  string `json:"flow_id"`
				Version int    `json:"version"`
			}
			_ = json.Unmarshal(request.RawParams(), &body)
			result, execErr := db.GetTaskFlowVersion(ctx, principal.WorkspaceID, body.FlowID, body.Version)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "flows.validate":
			var spec taskflow.Spec
			_ = json.Unmarshal(request.RawParams(), &spec)
			execErr := taskflow.Validate(spec)
			response, err = NewResponseFrame(request.ID, execErr == nil, map[string]bool{"ok": execErr == nil}, controlErrorText(execErr))
		case "flows.run":
			var body struct {
				FlowID string                 `json:"flow_id"`
				Inputs map[string]interface{} `json:"inputs"`
			}
			_ = json.Unmarshal(request.RawParams(), &body)
			result, execErr := s.TaskFlowEngine().Run(ctx, body.FlowID, body.Inputs)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "flow_runs.get":
			var body struct {
				RunID string `json:"run_id"`
			}
			_ = json.Unmarshal(request.RawParams(), &body)
			result, execErr := db.GetTaskFlowRun(ctx, principal.WorkspaceID, body.RunID)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		case "flow_runs.cancel":
			var body struct {
				RunID string `json:"run_id"`
			}
			_ = json.Unmarshal(request.RawParams(), &body)
			run, execErr := db.GetTaskFlowRun(ctx, principal.WorkspaceID, body.RunID)
			if execErr == nil && run != nil {
				run.Status = taskflow.StatusCancelled
				run.CompletedAt = time.Now().Unix()
				execErr = db.UpsertTaskFlowRun(ctx, *run)
			}
			response, err = NewResponseFrame(request.ID, execErr == nil, run, controlErrorText(execErr))
		case "flow_runs.retry_node":
			var body struct {
				RunID  string `json:"run_id"`
				NodeID string `json:"node_id"`
			}
			_ = json.Unmarshal(request.RawParams(), &body)
			result, execErr := s.TaskFlowEngine().RetryNode(ctx, body.RunID, body.NodeID)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, result, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		default:
			response, err = NewResponseFrame(request.ID, false, nil, "unsupported gateway method")
		}

		if err != nil {
			return
		}
		if err := conn.WriteJSON(response); err != nil {
			return
		}
		if response.OK {
			s.idempotency.Put(idempotencyKey, response, time.Now())
		}
	}
}

func controlErrorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func normalizeControlExecRequest(ctx context.Context, req *node.NodeCommandRequest) {
	if req == nil {
		return
	}
	if principal, ok := authz.PrincipalFromContext(ctx); ok {
		req.WorkspaceID = principal.WorkspaceID
		req.ActorID = principal.ActorID
		req.ActorRole = string(principal.Role)
		req.ActorScopes = append([]string(nil), principal.Scopes...)
		if req.UserID == "" {
			req.UserID = principal.ActorID
		}
	}
	req.RequireApproval = true
}
