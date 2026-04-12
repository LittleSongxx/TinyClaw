package http

import (
	stdhttp "net/http"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/doctor"
	"github.com/LittleSongxx/TinyClaw/gateway"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/plugins"
	"github.com/LittleSongxx/TinyClaw/taskflow"
	"github.com/LittleSongxx/TinyClaw/utils"
)

func DoctorRun(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	report := doctor.Run(r.Context(), doctor.Options{
		WorkspaceID: authz.WorkspaceIDFromContext(r.Context()),
		Fix:         r.URL.Query().Get("fix") == "true",
	})
	utils.Success(r.Context(), w, r, report)
}

func SecurityAudit(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	report := doctor.SecurityAudit(r.Context(), doctor.Options{
		WorkspaceID: authz.WorkspaceIDFromContext(r.Context()),
		Fix:         r.URL.Query().Get("fix") == "true",
	})
	utils.Success(r.Context(), w, r, report)
}

func DeviceBootstrap(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	result, err := gateway.DefaultService().CreateDeviceBootstrap(r.Context())
	respond(w, r, result, err)
}

func DevicePending(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	result, err := gateway.DefaultService().ListPendingDevices(r.Context())
	respond(w, r, result, err)
}

func DeviceApprove(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body struct {
		RequestID string `json:"request_id"`
	}
	if !decodeAdminJSON(w, r, &body) {
		return
	}
	result, err := gateway.DefaultService().ApproveDevice(r.Context(), body.RequestID)
	respond(w, r, result, err)
}

func DeviceReject(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body struct {
		RequestID string `json:"request_id"`
		Reason    string `json:"reason"`
	}
	if !decodeAdminJSON(w, r, &body) {
		return
	}
	err := gateway.DefaultService().RejectDevice(r.Context(), body.RequestID, body.Reason)
	respond(w, r, map[string]bool{"ok": err == nil}, err)
}

func DeviceRevoke(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body struct {
		DeviceID string `json:"device_id"`
	}
	if !decodeAdminJSON(w, r, &body) {
		return
	}
	err := gateway.DefaultService().RevokeDevice(r.Context(), body.DeviceID)
	respond(w, r, map[string]bool{"ok": err == nil}, err)
}

func PluginsList(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	result, err := gateway.DefaultService().PluginRegistry().List(r.Context())
	respond(w, r, result, err)
}

func PluginsStatus(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body struct {
		PluginID string `json:"plugin_id"`
	}
	if !decodeAdminJSON(w, r, &body) {
		return
	}
	result, err := gateway.DefaultService().PluginRegistry().Status(r.Context(), body.PluginID)
	respond(w, r, result, err)
}

func PluginsEnable(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	setPluginEnabled(w, r, true)
}

func PluginsDisable(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	setPluginEnabled(w, r, false)
}

func PluginsValidate(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var manifest plugins.Manifest
	if !decodeAdminJSON(w, r, &manifest) {
		return
	}
	err := gateway.DefaultService().PluginRegistry().Validate(r.Context(), manifest)
	respond(w, r, map[string]bool{"ok": err == nil}, err)
}

func setPluginEnabled(w stdhttp.ResponseWriter, r *stdhttp.Request, enabled bool) {
	var body struct {
		PluginID string `json:"plugin_id"`
		Config   string `json:"config"`
	}
	if !decodeAdminJSON(w, r, &body) {
		return
	}
	err := gateway.DefaultService().PluginRegistry().SetEnabled(r.Context(), body.PluginID, enabled, body.Config)
	respond(w, r, map[string]bool{"ok": err == nil}, err)
}

func FlowsCreate(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body struct {
		FlowID      string        `json:"flow_id"`
		Name        string        `json:"name"`
		Description string        `json:"description"`
		Spec        taskflow.Spec `json:"spec"`
	}
	if !decodeAdminJSON(w, r, &body) {
		return
	}
	result, err := taskflow.CreateOrUpdate(r.Context(), body.FlowID, body.Name, body.Description, body.Spec)
	respond(w, r, result, err)
}

func FlowsList(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	result, err := db.ListTaskFlows(r.Context(), authz.WorkspaceIDFromContext(r.Context()))
	respond(w, r, result, err)
}

func FlowsGet(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	flowID := r.URL.Query().Get("flow_id")
	version := 0
	result, err := db.GetTaskFlowVersion(r.Context(), authz.WorkspaceIDFromContext(r.Context()), flowID, version)
	respond(w, r, result, err)
}

func FlowsValidate(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var spec taskflow.Spec
	if !decodeAdminJSON(w, r, &spec) {
		return
	}
	err := taskflow.Validate(spec)
	respond(w, r, map[string]bool{"ok": err == nil}, err)
}

func FlowsRun(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body struct {
		FlowID string                 `json:"flow_id"`
		Inputs map[string]interface{} `json:"inputs"`
	}
	if !decodeAdminJSON(w, r, &body) {
		return
	}
	result, err := gateway.DefaultService().TaskFlowEngine().Run(r.Context(), body.FlowID, body.Inputs)
	respond(w, r, result, err)
}

func FlowRunGet(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	result, err := db.GetTaskFlowRun(r.Context(), authz.WorkspaceIDFromContext(r.Context()), r.URL.Query().Get("run_id"))
	respond(w, r, result, err)
}

func FlowRunCancel(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body struct {
		RunID string `json:"run_id"`
	}
	if !decodeAdminJSON(w, r, &body) {
		return
	}
	run, err := db.GetTaskFlowRun(r.Context(), authz.WorkspaceIDFromContext(r.Context()), body.RunID)
	if err == nil && run != nil {
		run.Status = taskflow.StatusCancelled
		run.CompletedAt = time.Now().Unix()
		err = db.UpsertTaskFlowRun(r.Context(), *run)
	}
	respond(w, r, run, err)
}

func FlowRunRetryNode(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body struct {
		RunID  string `json:"run_id"`
		NodeID string `json:"node_id"`
	}
	if !decodeAdminJSON(w, r, &body) {
		return
	}
	result, err := gateway.DefaultService().TaskFlowEngine().RetryNode(r.Context(), body.RunID, body.NodeID)
	respond(w, r, result, err)
}

func respond(w stdhttp.ResponseWriter, r *stdhttp.Request, payload interface{}, err error) {
	if err != nil {
		utils.Failure(r.Context(), w, r, param.CodeServerFail, err.Error(), err)
		return
	}
	utils.Success(r.Context(), w, r, payload)
}

func decodeAdminJSON(w stdhttp.ResponseWriter, r *stdhttp.Request, v interface{}) bool {
	if err := utils.HandleJsonBody(r, v); err != nil {
		utils.Failure(r.Context(), w, r, param.CodeParamError, param.MsgParamError, err)
		return false
	}
	return true
}
