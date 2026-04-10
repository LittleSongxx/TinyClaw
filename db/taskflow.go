package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/conf"
)

type TaskFlowRecord struct {
	FlowID         string `json:"flow_id"`
	WorkspaceID    string `json:"workspace_id"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	CurrentVersion int    `json:"current_version"`
	Status         string `json:"status"`
	CreateTime     int64  `json:"create_time"`
	UpdateTime     int64  `json:"update_time"`
}

type TaskFlowVersionRecord struct {
	FlowID      string                 `json:"flow_id"`
	WorkspaceID string                 `json:"workspace_id"`
	Version     int                    `json:"version"`
	Spec        map[string]interface{} `json:"spec"`
	CreateTime  int64                  `json:"create_time"`
}

type TaskFlowRunRecord struct {
	RunID       string                 `json:"run_id"`
	WorkspaceID string                 `json:"workspace_id"`
	FlowID      string                 `json:"flow_id"`
	Version     int                    `json:"version"`
	Status      string                 `json:"status"`
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
	Outputs     map[string]interface{} `json:"outputs,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreateTime  int64                  `json:"create_time"`
	UpdateTime  int64                  `json:"update_time"`
	CompletedAt int64                 `json:"completed_at,omitempty"`
}

type TaskFlowNodeRunRecord struct {
	ID          int64                  `json:"id"`
	WorkspaceID string                 `json:"workspace_id"`
	RunID       string                 `json:"run_id"`
	NodeID      string                 `json:"node_id"`
	NodeType    string                 `json:"node_type"`
	Status      string                 `json:"status"`
	Attempt     int                   `json:"attempt"`
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
	Outputs     map[string]interface{} `json:"outputs,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreateTime  int64                  `json:"create_time"`
	UpdateTime  int64                  `json:"update_time"`
	CompletedAt int64                 `json:"completed_at,omitempty"`
}

type TaskFlowEventRecord struct {
	ID          int64                  `json:"id"`
	WorkspaceID string                 `json:"workspace_id"`
	RunID       string                 `json:"run_id"`
	NodeID      string                 `json:"node_id,omitempty"`
	Event       string                 `json:"event"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	CreateTime  int64                  `json:"create_time"`
}

func UpsertTaskFlow(ctx context.Context, record TaskFlowRecord, spec map[string]interface{}) error {
	if DB == nil {
		return nil
	}
	record.WorkspaceID = authz.NormalizeWorkspaceID(record.WorkspaceID)
	if record.FlowID == "" {
		return fmt.Errorf("flow_id is required")
	}
	if record.CurrentVersion <= 0 {
		record.CurrentVersion = 1
	}
	if record.Status == "" {
		record.Status = "active"
	}
	now := time.Now().Unix()
	if record.CreateTime == 0 {
		record.CreateTime = now
	}
	record.UpdateTime = now
	if conf.BaseConfInfo.DBType == "mysql" {
		if _, err := DB.Exec(`
			INSERT INTO task_flows (flow_id, workspace_id, name, description, current_version, status, create_time, update_time)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE workspace_id = VALUES(workspace_id), name = VALUES(name), description = VALUES(description), current_version = VALUES(current_version), status = VALUES(status), update_time = VALUES(update_time)
		`, record.FlowID, record.WorkspaceID, record.Name, record.Description, record.CurrentVersion, record.Status, record.CreateTime, record.UpdateTime); err != nil {
			return err
		}
	} else if _, err := DB.Exec(`
		INSERT INTO task_flows (flow_id, workspace_id, name, description, current_version, status, create_time, update_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(flow_id) DO UPDATE SET workspace_id = excluded.workspace_id, name = excluded.name, description = excluded.description, current_version = excluded.current_version, status = excluded.status, update_time = excluded.update_time
	`, record.FlowID, record.WorkspaceID, record.Name, record.Description, record.CurrentVersion, record.Status, record.CreateTime, record.UpdateTime); err != nil {
		return err
	}
	return InsertTaskFlowVersion(ctx, TaskFlowVersionRecord{
		FlowID:      record.FlowID,
		WorkspaceID: record.WorkspaceID,
		Version:     record.CurrentVersion,
		Spec:        spec,
		CreateTime:  record.UpdateTime,
	})
}

func InsertTaskFlowVersion(ctx context.Context, version TaskFlowVersionRecord) error {
	if DB == nil {
		return nil
	}
	version.WorkspaceID = authz.NormalizeWorkspaceID(version.WorkspaceID)
	if version.FlowID == "" || version.Version <= 0 {
		return fmt.Errorf("flow_id and version are required")
	}
	body, _ := json.Marshal(version.Spec)
	if version.CreateTime == 0 {
		version.CreateTime = time.Now().Unix()
	}
	if conf.BaseConfInfo.DBType == "mysql" {
		_, err := DB.Exec(`
			INSERT INTO task_flow_versions (flow_id, workspace_id, version, spec, create_time)
			VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE spec = VALUES(spec), create_time = VALUES(create_time)
		`, version.FlowID, version.WorkspaceID, version.Version, string(body), version.CreateTime)
		return err
	}
	_, err := DB.Exec(`
		INSERT INTO task_flow_versions (flow_id, workspace_id, version, spec, create_time)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, flow_id, version) DO UPDATE SET spec = excluded.spec, create_time = excluded.create_time
	`, version.FlowID, version.WorkspaceID, version.Version, string(body), version.CreateTime)
	return err
}

func GetTaskFlow(ctx context.Context, workspaceID, flowID string) (*TaskFlowRecord, error) {
	if DB == nil {
		return nil, nil
	}
	return scanTaskFlow(DB.QueryRow(`
		SELECT flow_id, workspace_id, name, description, current_version, status, create_time, update_time
		FROM task_flows WHERE workspace_id = ? AND flow_id = ?
	`, authz.NormalizeWorkspaceID(workspaceID), strings.TrimSpace(flowID)))
}

func ListTaskFlows(ctx context.Context, workspaceID string) ([]TaskFlowRecord, error) {
	if DB == nil {
		return nil, nil
	}
	rows, err := DB.Query(`
		SELECT flow_id, workspace_id, name, description, current_version, status, create_time, update_time
		FROM task_flows WHERE workspace_id = ? ORDER BY update_time DESC
	`, authz.NormalizeWorkspaceID(workspaceID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TaskFlowRecord, 0)
	for rows.Next() {
		item, err := scanTaskFlow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func GetTaskFlowVersion(ctx context.Context, workspaceID, flowID string, version int) (*TaskFlowVersionRecord, error) {
	if DB == nil {
		return nil, nil
	}
	if version <= 0 {
		flow, err := GetTaskFlow(ctx, workspaceID, flowID)
		if err != nil || flow == nil {
			return nil, err
		}
		version = flow.CurrentVersion
	}
	return scanTaskFlowVersion(DB.QueryRow(`
		SELECT flow_id, workspace_id, version, spec, create_time
		FROM task_flow_versions WHERE workspace_id = ? AND flow_id = ? AND version = ?
	`, authz.NormalizeWorkspaceID(workspaceID), strings.TrimSpace(flowID), version))
}

func UpsertTaskFlowRun(ctx context.Context, record TaskFlowRunRecord) error {
	if DB == nil {
		return nil
	}
	record.WorkspaceID = authz.NormalizeWorkspaceID(record.WorkspaceID)
	if record.RunID == "" {
		return fmt.Errorf("run_id is required")
	}
	now := time.Now().Unix()
	if record.CreateTime == 0 {
		record.CreateTime = now
	}
	record.UpdateTime = now
	inputs, _ := json.Marshal(record.Inputs)
	outputs, _ := json.Marshal(record.Outputs)
	if conf.BaseConfInfo.DBType == "mysql" {
		_, err := DB.Exec(`
			INSERT INTO task_flow_runs (run_id, workspace_id, flow_id, version, status, inputs, outputs, error, create_time, update_time, completed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE status = VALUES(status), outputs = VALUES(outputs), error = VALUES(error), update_time = VALUES(update_time), completed_at = VALUES(completed_at)
		`, record.RunID, record.WorkspaceID, record.FlowID, record.Version, record.Status, string(inputs), string(outputs), record.Error, record.CreateTime, record.UpdateTime, record.CompletedAt)
		return err
	}
	_, err := DB.Exec(`
		INSERT INTO task_flow_runs (run_id, workspace_id, flow_id, version, status, inputs, outputs, error, create_time, update_time, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET status = excluded.status, outputs = excluded.outputs, error = excluded.error, update_time = excluded.update_time, completed_at = excluded.completed_at
	`, record.RunID, record.WorkspaceID, record.FlowID, record.Version, record.Status, string(inputs), string(outputs), record.Error, record.CreateTime, record.UpdateTime, record.CompletedAt)
	return err
}

func GetTaskFlowRun(ctx context.Context, workspaceID, runID string) (*TaskFlowRunRecord, error) {
	if DB == nil {
		return nil, nil
	}
	return scanTaskFlowRun(DB.QueryRow(`
		SELECT run_id, workspace_id, flow_id, version, status, inputs, outputs, error, create_time, update_time, completed_at
		FROM task_flow_runs WHERE workspace_id = ? AND run_id = ?
	`, authz.NormalizeWorkspaceID(workspaceID), strings.TrimSpace(runID)))
}

func UpsertTaskFlowNodeRun(ctx context.Context, record TaskFlowNodeRunRecord) error {
	if DB == nil {
		return nil
	}
	record.WorkspaceID = authz.NormalizeWorkspaceID(record.WorkspaceID)
	if record.RunID == "" || record.NodeID == "" {
		return fmt.Errorf("run_id and node_id are required")
	}
	now := time.Now().Unix()
	if record.CreateTime == 0 {
		record.CreateTime = now
	}
	record.UpdateTime = now
	inputs, _ := json.Marshal(record.Inputs)
	outputs, _ := json.Marshal(record.Outputs)
	if conf.BaseConfInfo.DBType == "mysql" {
		_, err := DB.Exec(`
			INSERT INTO task_flow_node_runs (workspace_id, run_id, node_id, node_type, status, attempt, inputs, outputs, error, create_time, update_time, completed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE status = VALUES(status), attempt = VALUES(attempt), inputs = VALUES(inputs), outputs = VALUES(outputs), error = VALUES(error), update_time = VALUES(update_time), completed_at = VALUES(completed_at)
		`, record.WorkspaceID, record.RunID, record.NodeID, record.NodeType, record.Status, record.Attempt, string(inputs), string(outputs), record.Error, record.CreateTime, record.UpdateTime, record.CompletedAt)
		return err
	}
	_, err := DB.Exec(`
		INSERT INTO task_flow_node_runs (workspace_id, run_id, node_id, node_type, status, attempt, inputs, outputs, error, create_time, update_time, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, run_id, node_id) DO UPDATE SET status = excluded.status, attempt = excluded.attempt, inputs = excluded.inputs, outputs = excluded.outputs, error = excluded.error, update_time = excluded.update_time, completed_at = excluded.completed_at
	`, record.WorkspaceID, record.RunID, record.NodeID, record.NodeType, record.Status, record.Attempt, string(inputs), string(outputs), record.Error, record.CreateTime, record.UpdateTime, record.CompletedAt)
	return err
}

func InsertTaskFlowEvent(ctx context.Context, event TaskFlowEventRecord) error {
	if DB == nil {
		return nil
	}
	event.WorkspaceID = authz.NormalizeWorkspaceID(event.WorkspaceID)
	if event.CreateTime == 0 {
		event.CreateTime = time.Now().Unix()
	}
	payload, _ := json.Marshal(event.Payload)
	_, err := DB.Exec(`
		INSERT INTO task_flow_events (workspace_id, run_id, node_id, event, payload, create_time)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.WorkspaceID, event.RunID, event.NodeID, event.Event, string(payload), event.CreateTime)
	return err
}

func scanTaskFlow(row rowScanner) (*TaskFlowRecord, error) {
	var item TaskFlowRecord
	if err := row.Scan(&item.FlowID, &item.WorkspaceID, &item.Name, &item.Description, &item.CurrentVersion, &item.Status, &item.CreateTime, &item.UpdateTime); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func scanTaskFlowVersion(row rowScanner) (*TaskFlowVersionRecord, error) {
	var (
		item TaskFlowVersionRecord
		specRaw string
	)
	if err := row.Scan(&item.FlowID, &item.WorkspaceID, &item.Version, &specRaw, &item.CreateTime); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	_ = json.Unmarshal([]byte(specRaw), &item.Spec)
	return &item, nil
}

func scanTaskFlowRun(row rowScanner) (*TaskFlowRunRecord, error) {
	var (
		item TaskFlowRunRecord
		inputsRaw string
		outputsRaw string
	)
	if err := row.Scan(&item.RunID, &item.WorkspaceID, &item.FlowID, &item.Version, &item.Status, &inputsRaw, &outputsRaw, &item.Error, &item.CreateTime, &item.UpdateTime, &item.CompletedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	_ = json.Unmarshal([]byte(inputsRaw), &item.Inputs)
	_ = json.Unmarshal([]byte(outputsRaw), &item.Outputs)
	return &item, nil
}
