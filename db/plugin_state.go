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

type PluginState struct {
	WorkspaceID string                 `json:"workspace_id"`
	PluginID    string                 `json:"plugin_id"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config,omitempty"`
	CreateTime  int64                  `json:"create_time"`
	UpdateTime  int64                  `json:"update_time"`
}

func UpsertPluginState(ctx context.Context, state PluginState) error {
	if DB == nil {
		return nil
	}
	state.WorkspaceID = authz.NormalizeWorkspaceID(state.WorkspaceID)
	state.PluginID = strings.TrimSpace(state.PluginID)
	if state.PluginID == "" {
		return fmt.Errorf("plugin_id is required")
	}
	body, _ := json.Marshal(state.Config)
	now := time.Now().Unix()
	if state.CreateTime == 0 {
		state.CreateTime = now
	}
	state.UpdateTime = now
	if conf.BaseConfInfo.DBType == "mysql" {
		_, err := DB.Exec(`
			INSERT INTO plugin_states (workspace_id, plugin_id, enabled, config, create_time, update_time)
			VALUES (?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE enabled = VALUES(enabled), config = VALUES(config), update_time = VALUES(update_time)
		`, state.WorkspaceID, state.PluginID, boolToInt(state.Enabled), string(body), state.CreateTime, state.UpdateTime)
		return err
	}
	_, err := DB.Exec(`
		INSERT INTO plugin_states (workspace_id, plugin_id, enabled, config, create_time, update_time)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, plugin_id) DO UPDATE SET enabled = excluded.enabled, config = excluded.config, update_time = excluded.update_time
	`, state.WorkspaceID, state.PluginID, boolToInt(state.Enabled), string(body), state.CreateTime, state.UpdateTime)
	return err
}

func GetPluginState(ctx context.Context, workspaceID, pluginID string) (*PluginState, error) {
	if DB == nil {
		return nil, nil
	}
	row := DB.QueryRow(`
		SELECT workspace_id, plugin_id, enabled, config, create_time, update_time
		FROM plugin_states WHERE workspace_id = ? AND plugin_id = ?
	`, authz.NormalizeWorkspaceID(workspaceID), strings.TrimSpace(pluginID))
	state, err := scanPluginState(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return state, nil
}

func ListPluginStates(ctx context.Context, workspaceID string) (map[string]PluginState, error) {
	if DB == nil {
		return nil, nil
	}
	rows, err := DB.Query(`
		SELECT workspace_id, plugin_id, enabled, config, create_time, update_time
		FROM plugin_states WHERE workspace_id = ?
	`, authz.NormalizeWorkspaceID(workspaceID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]PluginState)
	for rows.Next() {
		state, err := scanPluginState(rows)
		if err != nil {
			return nil, err
		}
		out[state.PluginID] = *state
	}
	return out, rows.Err()
}

func scanPluginState(row rowScanner) (*PluginState, error) {
	var (
		state     PluginState
		enabled   int
		configRaw string
	)
	if err := row.Scan(&state.WorkspaceID, &state.PluginID, &enabled, &configRaw, &state.CreateTime, &state.UpdateTime); err != nil {
		return nil, err
	}
	state.Enabled = enabled != 0
	_ = json.Unmarshal([]byte(configRaw), &state.Config)
	return &state, nil
}
