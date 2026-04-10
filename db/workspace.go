package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/conf"
)

const DefaultWorkspaceID = authz.DefaultWorkspaceID

type Workspace struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	CreateTime  int64  `json:"create_time"`
	UpdateTime  int64  `json:"update_time"`
}

type WorkspaceMembership struct {
	WorkspaceID string     `json:"workspace_id"`
	ActorID     string     `json:"actor_id"`
	Role        authz.Role `json:"role"`
	Scopes      []string   `json:"scopes,omitempty"`
	CreateTime  int64      `json:"create_time"`
	UpdateTime  int64      `json:"update_time"`
}

type AuditEvent struct {
	WorkspaceID  string                 `json:"workspace_id"`
	ActorID      string                 `json:"actor_id,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	Success      bool                   `json:"success"`
	Detail       string                 `json:"detail,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreateTime   int64                  `json:"create_time"`
}

func InitWorkspaceSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}
	if err := createWorkspaceTables(db); err != nil {
		return err
	}
	if err := ensureWorkspaceColumns(db); err != nil {
		return err
	}
	return EnsureDefaultWorkspace(context.Background())
}

func createWorkspaceTables(db *sql.DB) error {
	switch conf.BaseConfInfo.DBType {
	case "mysql":
		for i, sqlText := range mysqlWorkspaceSQLs {
			if _, err := db.Exec(sqlText); err != nil {
				return fmt.Errorf("create workspace mysql table batch %d fail: %w", i+1, err)
			}
		}
	default:
		for name, sqlText := range sqliteWorkspaceSQLs {
			if _, err := db.Exec(sqlText); err != nil {
				return fmt.Errorf("create workspace sqlite table %s fail: %w", name, err)
			}
		}
	}
	return nil
}

func ensureWorkspaceColumns(db *sql.DB) error {
	defs := map[string]string{
		"users":           "VARCHAR(100) NOT NULL DEFAULT 'default'",
		"records":         "VARCHAR(100) NOT NULL DEFAULT 'default'",
		"knowledge_files": "VARCHAR(100) NOT NULL DEFAULT 'default'",
		"cron":            "VARCHAR(100) NOT NULL DEFAULT 'default'",
		"agent_runs":      "VARCHAR(100) NOT NULL DEFAULT 'default'",
		"agent_steps":     "VARCHAR(100) NOT NULL DEFAULT 'default'",
		"sessions":        "VARCHAR(100) NOT NULL DEFAULT 'default'",
	}
	for tableName, definition := range defs {
		var err error
		if conf.BaseConfInfo.DBType == "mysql" {
			err = ensureMySQLColumn(db, tableName, "workspace_id", definition)
		} else {
			err = ensureSQLiteColumn(db, tableName, "workspace_id", definition)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func EnsureDefaultWorkspace(ctx context.Context) error {
	if DB == nil {
		return nil
	}
	now := time.Now().Unix()
	if conf.BaseConfInfo.DBType == "mysql" {
		if _, err := DB.Exec(`
			INSERT INTO workspaces (workspace_id, name, status, create_time, update_time)
			VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE name = VALUES(name), status = VALUES(status), update_time = VALUES(update_time)
		`, DefaultWorkspaceID, "Default Workspace", "active", now, now); err != nil {
			return err
		}
	} else {
		if _, err := DB.Exec(`
			INSERT INTO workspaces (workspace_id, name, status, create_time, update_time)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(workspace_id) DO UPDATE SET name = excluded.name, status = excluded.status, update_time = excluded.update_time
		`, DefaultWorkspaceID, "Default Workspace", "active", now, now); err != nil {
			return err
		}
	}

	owners := make([]string, 0)
	for actorID := range conf.BaseConfInfo.PrivilegedUserIds {
		actorID = strings.TrimSpace(actorID)
		if actorID != "" {
			owners = append(owners, actorID)
		}
	}
	if len(owners) == 0 {
		owners = append(owners, "system")
	}
	for _, actorID := range owners {
		if err := UpsertWorkspaceMembership(ctx, WorkspaceMembership{
			WorkspaceID: DefaultWorkspaceID,
			ActorID:     actorID,
			Role:        authz.RoleOwner,
			Scopes:      []string{"*"},
		}); err != nil {
			return err
		}
	}
	return nil
}

func UpsertWorkspaceMembership(ctx context.Context, membership WorkspaceMembership) error {
	if DB == nil {
		return nil
	}
	membership.WorkspaceID = authz.NormalizeWorkspaceID(membership.WorkspaceID)
	membership.ActorID = strings.TrimSpace(membership.ActorID)
	membership.Role = authz.NormalizeRole(membership.Role)
	membership.Scopes = authz.NormalizeScopes(membership.Scopes)
	if membership.ActorID == "" {
		return fmt.Errorf("actor_id is required")
	}
	body, _ := json.Marshal(membership.Scopes)
	now := time.Now().Unix()
	if conf.BaseConfInfo.DBType == "mysql" {
		_, err := DB.Exec(`
			INSERT INTO workspace_memberships (workspace_id, actor_id, role, scopes, create_time, update_time)
			VALUES (?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE role = VALUES(role), scopes = VALUES(scopes), update_time = VALUES(update_time)
		`, membership.WorkspaceID, membership.ActorID, membership.Role, string(body), now, now)
		return err
	}
	_, err := DB.Exec(`
		INSERT INTO workspace_memberships (workspace_id, actor_id, role, scopes, create_time, update_time)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, actor_id) DO UPDATE SET role = excluded.role, scopes = excluded.scopes, update_time = excluded.update_time
	`, membership.WorkspaceID, membership.ActorID, membership.Role, string(body), now, now)
	return err
}

func GetWorkspacePrincipal(ctx context.Context, workspaceID, actorID string) (authz.Principal, error) {
	workspaceID = authz.NormalizeWorkspaceID(workspaceID)
	actorID = strings.TrimSpace(actorID)
	if DB == nil || actorID == "" {
		return authz.Principal{}, authz.ErrMissingPrincipal
	}
	row := DB.QueryRow(`SELECT role, scopes FROM workspace_memberships WHERE workspace_id = ? AND actor_id = ?`, workspaceID, actorID)
	var (
		role      string
		scopesRaw string
	)
	if err := row.Scan(&role, &scopesRaw); err != nil {
		if err == sql.ErrNoRows {
			return authz.Principal{}, authz.ErrForbidden
		}
		return authz.Principal{}, err
	}
	var scopes []string
	_ = json.Unmarshal([]byte(scopesRaw), &scopes)
	return authz.NewPrincipal(workspaceID, actorID, authz.Role(role), scopes), nil
}

func InsertAuditEvent(ctx context.Context, event AuditEvent) error {
	if DB == nil {
		return nil
	}
	event.WorkspaceID = authz.NormalizeWorkspaceID(event.WorkspaceID)
	if event.WorkspaceID == "" {
		event.WorkspaceID = authz.WorkspaceIDFromContext(ctx)
	}
	if event.CreateTime == 0 {
		event.CreateTime = time.Now().Unix()
	}
	metadata, _ := json.Marshal(event.Metadata)
	_, err := DB.Exec(`
		INSERT INTO audit_events (workspace_id, actor_id, action, resource_type, resource_id, success, detail, metadata, create_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.WorkspaceID, event.ActorID, event.Action, event.ResourceType, event.ResourceID, boolToInt(event.Success), event.Detail, string(metadata), event.CreateTime)
	return err
}

func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(secret)))
	return hex.EncodeToString(sum[:])
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func normalizeWorkspaceID(workspaceID string) string {
	return authz.NormalizeWorkspaceID(workspaceID)
}

var sqliteWorkspaceSQLs = map[string]string{
	"workspaces": `
		CREATE TABLE IF NOT EXISTS workspaces (
			workspace_id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL DEFAULT '',
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			create_time INTEGER NOT NULL DEFAULT 0,
			update_time INTEGER NOT NULL DEFAULT 0
		);`,
	"workspace_memberships": `
		CREATE TABLE IF NOT EXISTS workspace_memberships (
			workspace_id VARCHAR(100) NOT NULL,
			actor_id VARCHAR(255) NOT NULL,
			role VARCHAR(50) NOT NULL DEFAULT 'viewer',
			scopes TEXT NOT NULL DEFAULT '[]',
			create_time INTEGER NOT NULL DEFAULT 0,
			update_time INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (workspace_id, actor_id)
		);`,
	"api_tokens": `
		CREATE TABLE IF NOT EXISTS api_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
			token_hash VARCHAR(128) NOT NULL DEFAULT '',
			name VARCHAR(255) NOT NULL DEFAULT '',
			actor_id VARCHAR(255) NOT NULL DEFAULT '',
			role VARCHAR(50) NOT NULL DEFAULT 'viewer',
			scopes TEXT NOT NULL DEFAULT '[]',
			revoked_at INTEGER NOT NULL DEFAULT 0,
			create_time INTEGER NOT NULL DEFAULT 0,
			update_time INTEGER NOT NULL DEFAULT 0
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_api_tokens_hash ON api_tokens(token_hash);`,
	"audit_events": `
		CREATE TABLE IF NOT EXISTS audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
			actor_id VARCHAR(255) NOT NULL DEFAULT '',
			action VARCHAR(255) NOT NULL DEFAULT '',
			resource_type VARCHAR(100) NOT NULL DEFAULT '',
			resource_id VARCHAR(255) NOT NULL DEFAULT '',
			success INTEGER NOT NULL DEFAULT 0,
			detail TEXT NOT NULL DEFAULT '',
			metadata TEXT NOT NULL DEFAULT '{}',
			create_time INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_audit_events_workspace ON audit_events(workspace_id, create_time);`,
	"devices": `
		CREATE TABLE IF NOT EXISTS devices (
			device_id VARCHAR(255) PRIMARY KEY,
			workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
			device_token_hash VARCHAR(128) NOT NULL DEFAULT '',
			public_key TEXT NOT NULL DEFAULT '',
			name VARCHAR(255) NOT NULL DEFAULT '',
			platform VARCHAR(100) NOT NULL DEFAULT '',
			device_family VARCHAR(100) NOT NULL DEFAULT '',
			metadata TEXT NOT NULL DEFAULT '{}',
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			create_time INTEGER NOT NULL DEFAULT 0,
			update_time INTEGER NOT NULL DEFAULT 0,
			last_seen_at INTEGER NOT NULL DEFAULT 0,
			revoked_at INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_devices_workspace ON devices(workspace_id, status);`,
	"device_pairing_requests": `
		CREATE TABLE IF NOT EXISTS device_pairing_requests (
			request_id VARCHAR(100) PRIMARY KEY,
			workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
			bootstrap_token_hash VARCHAR(128) NOT NULL DEFAULT '',
			bootstrap_code_hash VARCHAR(128) NOT NULL DEFAULT '',
			device_id VARCHAR(255) NOT NULL DEFAULT '',
			public_key TEXT NOT NULL DEFAULT '',
			descriptor TEXT NOT NULL DEFAULT '{}',
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			expires_at INTEGER NOT NULL DEFAULT 0,
			issued_token_hash VARCHAR(128) NOT NULL DEFAULT '',
			create_time INTEGER NOT NULL DEFAULT 0,
			update_time INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_device_pairing_workspace ON device_pairing_requests(workspace_id, status);`,
	"plugin_states": `
		CREATE TABLE IF NOT EXISTS plugin_states (
			workspace_id VARCHAR(100) NOT NULL,
			plugin_id VARCHAR(255) NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 0,
			config TEXT NOT NULL DEFAULT '{}',
			create_time INTEGER NOT NULL DEFAULT 0,
			update_time INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (workspace_id, plugin_id)
		);`,
	"task_flows": `
		CREATE TABLE IF NOT EXISTS task_flows (
			flow_id VARCHAR(100) PRIMARY KEY,
			workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
			name VARCHAR(255) NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			current_version INTEGER NOT NULL DEFAULT 0,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			create_time INTEGER NOT NULL DEFAULT 0,
			update_time INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_task_flows_workspace ON task_flows(workspace_id, update_time);`,
	"task_flow_versions": `
		CREATE TABLE IF NOT EXISTS task_flow_versions (
			flow_id VARCHAR(100) NOT NULL,
			workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
			version INTEGER NOT NULL DEFAULT 1,
			spec TEXT NOT NULL DEFAULT '{}',
			create_time INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (workspace_id, flow_id, version)
		);`,
	"task_flow_runs": `
		CREATE TABLE IF NOT EXISTS task_flow_runs (
			run_id VARCHAR(100) PRIMARY KEY,
			workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
			flow_id VARCHAR(100) NOT NULL DEFAULT '',
			version INTEGER NOT NULL DEFAULT 0,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			inputs TEXT NOT NULL DEFAULT '{}',
			outputs TEXT NOT NULL DEFAULT '{}',
			error TEXT NOT NULL DEFAULT '',
			create_time INTEGER NOT NULL DEFAULT 0,
			update_time INTEGER NOT NULL DEFAULT 0,
			completed_at INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_task_flow_runs_workspace ON task_flow_runs(workspace_id, flow_id, update_time);`,
	"task_flow_node_runs": `
		CREATE TABLE IF NOT EXISTS task_flow_node_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
			run_id VARCHAR(100) NOT NULL DEFAULT '',
			node_id VARCHAR(100) NOT NULL DEFAULT '',
			node_type VARCHAR(50) NOT NULL DEFAULT '',
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			attempt INTEGER NOT NULL DEFAULT 0,
			inputs TEXT NOT NULL DEFAULT '{}',
			outputs TEXT NOT NULL DEFAULT '{}',
			error TEXT NOT NULL DEFAULT '',
			create_time INTEGER NOT NULL DEFAULT 0,
			update_time INTEGER NOT NULL DEFAULT 0,
			completed_at INTEGER NOT NULL DEFAULT 0,
			UNIQUE (workspace_id, run_id, node_id)
		);
		CREATE INDEX IF NOT EXISTS idx_task_flow_node_runs_run ON task_flow_node_runs(workspace_id, run_id);`,
	"task_flow_events": `
		CREATE TABLE IF NOT EXISTS task_flow_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
			run_id VARCHAR(100) NOT NULL DEFAULT '',
			node_id VARCHAR(100) NOT NULL DEFAULT '',
			event VARCHAR(100) NOT NULL DEFAULT '',
			payload TEXT NOT NULL DEFAULT '{}',
			create_time INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_task_flow_events_run ON task_flow_events(workspace_id, run_id, id);`,
}

var mysqlWorkspaceSQLs = []string{
	`CREATE TABLE IF NOT EXISTS workspaces (
		workspace_id VARCHAR(100) NOT NULL PRIMARY KEY,
		name VARCHAR(255) NOT NULL DEFAULT '',
		status VARCHAR(50) NOT NULL DEFAULT 'active',
		create_time INT(10) NOT NULL DEFAULT 0,
		update_time INT(10) NOT NULL DEFAULT 0
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	`CREATE TABLE IF NOT EXISTS workspace_memberships (
		workspace_id VARCHAR(100) NOT NULL,
		actor_id VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'viewer',
		scopes MEDIUMTEXT NOT NULL,
		create_time INT(10) NOT NULL DEFAULT 0,
		update_time INT(10) NOT NULL DEFAULT 0,
		PRIMARY KEY (workspace_id, actor_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	`CREATE TABLE IF NOT EXISTS api_tokens (
		id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
		token_hash VARCHAR(128) NOT NULL DEFAULT '',
		name VARCHAR(255) NOT NULL DEFAULT '',
		actor_id VARCHAR(255) NOT NULL DEFAULT '',
		role VARCHAR(50) NOT NULL DEFAULT 'viewer',
		scopes MEDIUMTEXT NOT NULL,
		revoked_at INT(10) NOT NULL DEFAULT 0,
		create_time INT(10) NOT NULL DEFAULT 0,
		update_time INT(10) NOT NULL DEFAULT 0,
		UNIQUE KEY idx_api_tokens_hash (token_hash)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	`CREATE TABLE IF NOT EXISTS audit_events (
		id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
		actor_id VARCHAR(255) NOT NULL DEFAULT '',
		action VARCHAR(255) NOT NULL DEFAULT '',
		resource_type VARCHAR(100) NOT NULL DEFAULT '',
		resource_id VARCHAR(255) NOT NULL DEFAULT '',
		success TINYINT(1) NOT NULL DEFAULT 0,
		detail MEDIUMTEXT NOT NULL,
		metadata MEDIUMTEXT NOT NULL,
		create_time INT(10) NOT NULL DEFAULT 0,
		INDEX idx_audit_events_workspace (workspace_id, create_time)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	`CREATE TABLE IF NOT EXISTS devices (
		device_id VARCHAR(255) NOT NULL PRIMARY KEY,
		workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
		device_token_hash VARCHAR(128) NOT NULL DEFAULT '',
		public_key MEDIUMTEXT NOT NULL,
		name VARCHAR(255) NOT NULL DEFAULT '',
		platform VARCHAR(100) NOT NULL DEFAULT '',
		device_family VARCHAR(100) NOT NULL DEFAULT '',
		metadata MEDIUMTEXT NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'active',
		create_time INT(10) NOT NULL DEFAULT 0,
		update_time INT(10) NOT NULL DEFAULT 0,
		last_seen_at INT(10) NOT NULL DEFAULT 0,
		revoked_at INT(10) NOT NULL DEFAULT 0,
		INDEX idx_devices_workspace (workspace_id, status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	`CREATE TABLE IF NOT EXISTS device_pairing_requests (
		request_id VARCHAR(100) NOT NULL PRIMARY KEY,
		workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
		bootstrap_token_hash VARCHAR(128) NOT NULL DEFAULT '',
		bootstrap_code_hash VARCHAR(128) NOT NULL DEFAULT '',
		device_id VARCHAR(255) NOT NULL DEFAULT '',
		public_key MEDIUMTEXT NOT NULL,
		descriptor MEDIUMTEXT NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		expires_at INT(10) NOT NULL DEFAULT 0,
		issued_token_hash VARCHAR(128) NOT NULL DEFAULT '',
		create_time INT(10) NOT NULL DEFAULT 0,
		update_time INT(10) NOT NULL DEFAULT 0,
		INDEX idx_device_pairing_workspace (workspace_id, status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	`CREATE TABLE IF NOT EXISTS plugin_states (
		workspace_id VARCHAR(100) NOT NULL,
		plugin_id VARCHAR(255) NOT NULL,
		enabled TINYINT(1) NOT NULL DEFAULT 0,
		config MEDIUMTEXT NOT NULL,
		create_time INT(10) NOT NULL DEFAULT 0,
		update_time INT(10) NOT NULL DEFAULT 0,
		PRIMARY KEY (workspace_id, plugin_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	`CREATE TABLE IF NOT EXISTS task_flows (
		flow_id VARCHAR(100) NOT NULL PRIMARY KEY,
		workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
		name VARCHAR(255) NOT NULL DEFAULT '',
		description MEDIUMTEXT NOT NULL,
		current_version INT NOT NULL DEFAULT 0,
		status VARCHAR(50) NOT NULL DEFAULT 'active',
		create_time INT(10) NOT NULL DEFAULT 0,
		update_time INT(10) NOT NULL DEFAULT 0,
		INDEX idx_task_flows_workspace (workspace_id, update_time)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	`CREATE TABLE IF NOT EXISTS task_flow_versions (
		flow_id VARCHAR(100) NOT NULL,
		workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
		version INT NOT NULL DEFAULT 1,
		spec MEDIUMTEXT NOT NULL,
		create_time INT(10) NOT NULL DEFAULT 0,
		PRIMARY KEY (workspace_id, flow_id, version)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	`CREATE TABLE IF NOT EXISTS task_flow_runs (
		run_id VARCHAR(100) NOT NULL PRIMARY KEY,
		workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
		flow_id VARCHAR(100) NOT NULL DEFAULT '',
		version INT NOT NULL DEFAULT 0,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		inputs MEDIUMTEXT NOT NULL,
		outputs MEDIUMTEXT NOT NULL,
		error MEDIUMTEXT NOT NULL,
		create_time INT(10) NOT NULL DEFAULT 0,
		update_time INT(10) NOT NULL DEFAULT 0,
		completed_at INT(10) NOT NULL DEFAULT 0,
		INDEX idx_task_flow_runs_workspace (workspace_id, flow_id, update_time)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	`CREATE TABLE IF NOT EXISTS task_flow_node_runs (
		id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
		run_id VARCHAR(100) NOT NULL DEFAULT '',
		node_id VARCHAR(100) NOT NULL DEFAULT '',
		node_type VARCHAR(50) NOT NULL DEFAULT '',
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		attempt INT NOT NULL DEFAULT 0,
		inputs MEDIUMTEXT NOT NULL,
		outputs MEDIUMTEXT NOT NULL,
		error MEDIUMTEXT NOT NULL,
		create_time INT(10) NOT NULL DEFAULT 0,
		update_time INT(10) NOT NULL DEFAULT 0,
		completed_at INT(10) NOT NULL DEFAULT 0,
		UNIQUE KEY idx_task_flow_node_unique (workspace_id, run_id, node_id),
		INDEX idx_task_flow_node_runs_run (workspace_id, run_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	`CREATE TABLE IF NOT EXISTS task_flow_events (
		id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
		run_id VARCHAR(100) NOT NULL DEFAULT '',
		node_id VARCHAR(100) NOT NULL DEFAULT '',
		event VARCHAR(100) NOT NULL DEFAULT '',
		payload MEDIUMTEXT NOT NULL,
		create_time INT(10) NOT NULL DEFAULT 0,
		INDEX idx_task_flow_events_run (workspace_id, run_id, id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
}
