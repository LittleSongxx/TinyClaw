package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/tooling"
)

type AgentRun struct {
	ID          int64  `json:"id"`
	UserId      string `json:"user_id"`
	ChatId      string `json:"chat_id"`
	MsgId       string `json:"msg_id"`
	Mode        string `json:"mode"`
	Input       string `json:"input"`
	FinalOutput string `json:"final_output"`
	Status      string `json:"status"`
	Error       string `json:"error"`
	TokenTotal  int    `json:"token_total"`
	StepCount   int    `json:"step_count"`
	ReplayOf    int64  `json:"replay_of"`
	CreateTime  int64  `json:"create_time"`
	UpdateTime  int64  `json:"update_time"`
}

type AgentStep struct {
	ID           int64                 `json:"id"`
	RunID        int64                 `json:"run_id"`
	StepIndex    int                   `json:"step_index"`
	Kind         string                `json:"kind"`
	Name         string                `json:"name"`
	ToolName     string                `json:"tool_name"`
	Input        string                `json:"input"`
	RawOutput    string                `json:"raw_output"`
	Observations []tooling.Observation `json:"observations,omitempty"`
	Token        int                   `json:"token"`
	Status       string                `json:"status"`
	Error        string                `json:"error"`
	Provider     string                `json:"provider"`
	Model        string                `json:"model"`
	CreateTime   int64                 `json:"create_time"`
	UpdateTime   int64                 `json:"update_time"`
}

type AgentRunDetail struct {
	Run   *AgentRun   `json:"run"`
	Steps []AgentStep `json:"steps"`
}

func InsertAgentRun(run *AgentRun) (int64, error) {
	if run == nil {
		return 0, fmt.Errorf("run is nil")
	}

	now := time.Now().Unix()
	run.CreateTime = now
	run.UpdateTime = now

	query := `INSERT INTO agent_runs (user_id, chat_id, msg_id, mode, input, final_output, status, error, token_total, step_count, replay_of, create_time, update_time, from_bot)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []interface{}{
		run.UserId, run.ChatId, run.MsgId, run.Mode, run.Input, run.FinalOutput, run.Status, run.Error,
		run.TokenTotal, run.StepCount, run.ReplayOf, now, now, conf.BaseConfInfo.BotName,
	}

	if FeatureEnabled() {
		query = `INSERT INTO agent_runs (user_id, chat_id, msg_id, mode, input, final_output, status, error, token_total, step_count, replay_of, create_time, update_time, from_bot)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id`
		var id int64
		err := FeatureDB.QueryRow(query, args...).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}

	result, err := DB.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func UpdateAgentRun(run *AgentRun) error {
	if run == nil || run.ID == 0 {
		return fmt.Errorf("run is invalid")
	}

	run.UpdateTime = time.Now().Unix()
	query := `UPDATE agent_runs SET user_id = ?, chat_id = ?, msg_id = ?, mode = ?, input = ?, final_output = ?, status = ?, error = ?, token_total = ?, step_count = ?, replay_of = ?, update_time = ?
		WHERE id = ? AND from_bot = ?`
	args := []interface{}{
		run.UserId, run.ChatId, run.MsgId, run.Mode, run.Input, run.FinalOutput, run.Status, run.Error,
		run.TokenTotal, run.StepCount, run.ReplayOf, run.UpdateTime, run.ID, conf.BaseConfInfo.BotName,
	}

	if FeatureEnabled() {
		query = `UPDATE agent_runs SET user_id = $1, chat_id = $2, msg_id = $3, mode = $4, input = $5, final_output = $6, status = $7, error = $8, token_total = $9, step_count = $10, replay_of = $11, update_time = $12
			WHERE id = $13 AND from_bot = $14`
		_, err := FeatureDB.Exec(query, args...)
		return err
	}

	_, err := DB.Exec(query, args...)
	return err
}

func GetAgentRunByID(id int64) (*AgentRun, error) {
	query := `SELECT id, user_id, chat_id, msg_id, mode, input, final_output, status, error, token_total, step_count, replay_of, create_time, update_time
		FROM agent_runs WHERE id = ? AND from_bot = ?`
	args := []interface{}{id, conf.BaseConfInfo.BotName}

	var row *sql.Row
	if FeatureEnabled() {
		query = `SELECT id, user_id, chat_id, msg_id, mode, input, final_output, status, error, token_total, step_count, replay_of, create_time, update_time
			FROM agent_runs WHERE id = $1 AND from_bot = $2`
		row = FeatureDB.QueryRow(query, args...)
	} else {
		row = DB.QueryRow(query, args...)
	}

	var run AgentRun
	if err := row.Scan(&run.ID, &run.UserId, &run.ChatId, &run.MsgId, &run.Mode, &run.Input, &run.FinalOutput,
		&run.Status, &run.Error, &run.TokenTotal, &run.StepCount, &run.ReplayOf, &run.CreateTime, &run.UpdateTime); err != nil {
		return nil, err
	}
	return &run, nil
}

func GetAgentRunDetailByID(id int64) (*AgentRunDetail, error) {
	run, err := GetAgentRunByID(id)
	if err != nil {
		return nil, err
	}

	steps, err := GetAgentStepsByRunID(id)
	if err != nil {
		return nil, err
	}

	return &AgentRunDetail{Run: run, Steps: steps}, nil
}

func GetAgentRunsByPage(page, pageSize int, mode, status, userId string) ([]AgentRun, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	whereSQL := "WHERE from_bot = ?"
	args := []interface{}{conf.BaseConfInfo.BotName}

	if mode != "" {
		whereSQL += " AND mode = ?"
		args = append(args, mode)
	}
	if status != "" {
		whereSQL += " AND status = ?"
		args = append(args, status)
	}
	if userId != "" {
		whereSQL += " AND user_id = ?"
		args = append(args, userId)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, chat_id, msg_id, mode, input, final_output, status, error, token_total, step_count, replay_of, create_time, update_time
		FROM agent_runs %s
		ORDER BY id DESC
		LIMIT ? OFFSET ?`, whereSQL)
	args = append(args, pageSize, offset)

	var (
		rows *sql.Rows
		err  error
	)
	if FeatureEnabled() {
		whereSQL = "WHERE from_bot = $1"
		args = []interface{}{conf.BaseConfInfo.BotName}
		index := 2
		if mode != "" {
			whereSQL += fmt.Sprintf(" AND mode = $%d", index)
			args = append(args, mode)
			index++
		}
		if status != "" {
			whereSQL += fmt.Sprintf(" AND status = $%d", index)
			args = append(args, status)
			index++
		}
		if userId != "" {
			whereSQL += fmt.Sprintf(" AND user_id = $%d", index)
			args = append(args, userId)
			index++
		}
		query = fmt.Sprintf(`
			SELECT id, user_id, chat_id, msg_id, mode, input, final_output, status, error, token_total, step_count, replay_of, create_time, update_time
			FROM agent_runs %s
			ORDER BY id DESC
			LIMIT $%d OFFSET $%d`, whereSQL, index, index+1)
		args = append(args, pageSize, offset)
		rows, err = FeatureDB.Query(query, args...)
	} else {
		rows, err = DB.Query(query, args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]AgentRun, 0)
	for rows.Next() {
		var run AgentRun
		if err := rows.Scan(&run.ID, &run.UserId, &run.ChatId, &run.MsgId, &run.Mode, &run.Input, &run.FinalOutput,
			&run.Status, &run.Error, &run.TokenTotal, &run.StepCount, &run.ReplayOf, &run.CreateTime, &run.UpdateTime); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}

	return runs, rows.Err()
}

func GetAgentRunsCount(mode, status, userId string) (int, error) {
	whereSQL := "WHERE from_bot = ?"
	args := []interface{}{conf.BaseConfInfo.BotName}

	if mode != "" {
		whereSQL += " AND mode = ?"
		args = append(args, mode)
	}
	if status != "" {
		whereSQL += " AND status = ?"
		args = append(args, status)
	}
	if userId != "" {
		whereSQL += " AND user_id = ?"
		args = append(args, userId)
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM agent_runs %s", whereSQL)
	var count int
	if FeatureEnabled() {
		whereSQL = "WHERE from_bot = $1"
		args = []interface{}{conf.BaseConfInfo.BotName}
		index := 2
		if mode != "" {
			whereSQL += fmt.Sprintf(" AND mode = $%d", index)
			args = append(args, mode)
			index++
		}
		if status != "" {
			whereSQL += fmt.Sprintf(" AND status = $%d", index)
			args = append(args, status)
			index++
		}
		if userId != "" {
			whereSQL += fmt.Sprintf(" AND user_id = $%d", index)
			args = append(args, userId)
			index++
		}
		query = fmt.Sprintf("SELECT COUNT(*) FROM agent_runs %s", whereSQL)
		err := FeatureDB.QueryRow(query, args...).Scan(&count)
		return count, err
	}

	err := DB.QueryRow(query, args...).Scan(&count)
	return count, err
}

func InsertAgentStep(step *AgentStep) (int64, error) {
	if step == nil {
		return 0, fmt.Errorf("step is nil")
	}

	obs, err := marshalObservations(step.Observations)
	if err != nil {
		return 0, err
	}

	now := time.Now().Unix()
	step.CreateTime = now
	step.UpdateTime = now
	query := `INSERT INTO agent_steps (run_id, step_index, kind, name, tool_name, input, raw_output, observations, token, status, error, provider, model, create_time, update_time, from_bot)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []interface{}{
		step.RunID, step.StepIndex, step.Kind, step.Name, step.ToolName, step.Input, step.RawOutput, obs,
		step.Token, step.Status, step.Error, step.Provider, step.Model, now, now, conf.BaseConfInfo.BotName,
	}

	if FeatureEnabled() {
		query = `INSERT INTO agent_steps (run_id, step_index, kind, name, tool_name, input, raw_output, observations, token, status, error, provider, model, create_time, update_time, from_bot)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING id`
		var id int64
		err := FeatureDB.QueryRow(query, args...).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}

	result, err := DB.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func UpdateAgentStep(step *AgentStep) error {
	if step == nil || step.ID == 0 {
		return fmt.Errorf("step is invalid")
	}

	obs, err := marshalObservations(step.Observations)
	if err != nil {
		return err
	}

	step.UpdateTime = time.Now().Unix()
	query := `UPDATE agent_steps SET run_id = ?, step_index = ?, kind = ?, name = ?, tool_name = ?, input = ?, raw_output = ?, observations = ?, token = ?, status = ?, error = ?, provider = ?, model = ?, update_time = ?
		WHERE id = ? AND from_bot = ?`
	args := []interface{}{
		step.RunID, step.StepIndex, step.Kind, step.Name, step.ToolName, step.Input, step.RawOutput, obs,
		step.Token, step.Status, step.Error, step.Provider, step.Model, step.UpdateTime, step.ID, conf.BaseConfInfo.BotName,
	}

	if FeatureEnabled() {
		query = `UPDATE agent_steps SET run_id = $1, step_index = $2, kind = $3, name = $4, tool_name = $5, input = $6, raw_output = $7, observations = $8::jsonb, token = $9, status = $10, error = $11, provider = $12, model = $13, update_time = $14
			WHERE id = $15 AND from_bot = $16`
		_, err := FeatureDB.Exec(query, args...)
		return err
	}

	_, err = DB.Exec(query, args...)
	return err
}

func GetAgentStepsByRunID(runID int64) ([]AgentStep, error) {
	query := `SELECT id, run_id, step_index, kind, name, tool_name, input, raw_output, observations, token, status, error, provider, model, create_time, update_time
		FROM agent_steps WHERE run_id = ? AND from_bot = ? ORDER BY step_index ASC, id ASC`
	args := []interface{}{runID, conf.BaseConfInfo.BotName}

	var (
		rows *sql.Rows
		err  error
	)
	if FeatureEnabled() {
		query = `SELECT id, run_id, step_index, kind, name, tool_name, input, raw_output, observations, token, status, error, provider, model, create_time, update_time
			FROM agent_steps WHERE run_id = $1 AND from_bot = $2 ORDER BY step_index ASC, id ASC`
		rows, err = FeatureDB.Query(query, args...)
	} else {
		rows, err = DB.Query(query, args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	steps := make([]AgentStep, 0)
	for rows.Next() {
		var (
			step            AgentStep
			observationsRaw string
		)
		if err := rows.Scan(&step.ID, &step.RunID, &step.StepIndex, &step.Kind, &step.Name, &step.ToolName, &step.Input,
			&step.RawOutput, &observationsRaw, &step.Token, &step.Status, &step.Error, &step.Provider, &step.Model, &step.CreateTime, &step.UpdateTime); err != nil {
			return nil, err
		}

		step.Observations, err = unmarshalObservations(observationsRaw)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}

	return steps, rows.Err()
}

func marshalObservations(observations []tooling.Observation) (string, error) {
	if len(observations) == 0 {
		return "[]", nil
	}

	body, err := json.Marshal(observations)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func unmarshalObservations(raw string) ([]tooling.Observation, error) {
	if raw == "" {
		return nil, nil
	}

	observations := make([]tooling.Observation, 0)
	if err := json.Unmarshal([]byte(raw), &observations); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return observations, nil
}
