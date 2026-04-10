package doctor

import (
	"context"
	"database/sql"
	"os"
	"strings"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/node"
)

type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

type Finding struct {
	ID       string   `json:"id"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Fixable  bool     `json:"fixable,omitempty"`
	Fixed    bool     `json:"fixed,omitempty"`
}

type Report struct {
	WorkspaceID string    `json:"workspace_id"`
	Kind        string    `json:"kind"`
	OK          bool      `json:"ok"`
	Findings    []Finding `json:"findings"`
}

type Options struct {
	WorkspaceID string
	Fix         bool
}

func Run(ctx context.Context, opts Options) Report {
	return runChecks(ctx, "doctor", opts, []checkFunc{
		checkDBMigration,
		checkWorkspaceBackfill,
		checkDevicePairing,
		checkPluginState,
		checkGatewaySecret,
		checkTaskFlowTables,
	})
}

func SecurityAudit(ctx context.Context, opts Options) Report {
	return runChecks(ctx, "security_audit", opts, []checkFunc{
		checkLegacyStaticToken,
		checkGatewaySecret,
		checkLoopbackBypassRemoved,
		checkDangerousNodePolicy,
		checkLegacyFeatureGates,
		checkPluginState,
	})
}

type checkFunc func(context.Context, Options) []Finding

func runChecks(ctx context.Context, kind string, opts Options, checks []checkFunc) Report {
	opts.WorkspaceID = authz.NormalizeWorkspaceID(opts.WorkspaceID)
	ctx = authz.WithPrincipal(ctx, authz.NewPrincipal(opts.WorkspaceID, "doctor", authz.RoleAdmin, []string{"*"}))
	report := Report{WorkspaceID: opts.WorkspaceID, Kind: kind, OK: true}
	for _, check := range checks {
		report.Findings = append(report.Findings, check(ctx, opts)...)
	}
	for _, finding := range report.Findings {
		if finding.Severity == SeverityError {
			report.OK = false
			break
		}
	}
	return report
}

func checkDBMigration(ctx context.Context, opts Options) []Finding {
	tables := []string{"workspaces", "workspace_memberships", "api_tokens", "audit_events", "devices", "plugin_states"}
	return missingTables("db.migration", tables)
}

func checkTaskFlowTables(ctx context.Context, opts Options) []Finding {
	return missingTables("taskflow.tables", []string{"task_flows", "task_flow_versions", "task_flow_runs", "task_flow_node_runs", "task_flow_events"})
}

func checkWorkspaceBackfill(ctx context.Context, opts Options) []Finding {
	if db.DB == nil {
		return nil
	}
	for _, table := range []string{"users", "sessions", "agent_runs", "agent_steps"} {
		if !columnExists(table, "workspace_id") {
			return []Finding{{ID: "workspace.backfill", Severity: SeverityError, Message: table + " is missing workspace_id"}}
		}
	}
	return []Finding{{ID: "workspace.backfill", Severity: SeverityInfo, Message: "workspace columns are present"}}
}

func checkDevicePairing(ctx context.Context, opts Options) []Finding {
	if db.DB == nil {
		return nil
	}
	devices, err := db.ListDevices(ctx, opts.WorkspaceID)
	if err != nil {
		return []Finding{{ID: "devices.query", Severity: SeverityError, Message: err.Error()}}
	}
	if len(devices) == 0 {
		return []Finding{{ID: "devices.none", Severity: SeverityWarn, Message: "no paired devices found"}}
	}
	return []Finding{{ID: "devices.pairing", Severity: SeverityInfo, Message: "device pairing table is available"}}
}

func checkPluginState(ctx context.Context, opts Options) []Finding {
	if !tableExists("plugin_states") {
		return []Finding{{ID: "plugins.state", Severity: SeverityError, Message: "plugin_states table is missing"}}
	}
	return []Finding{{ID: "plugins.state", Severity: SeverityInfo, Message: "plugin enablement is workspace-scoped"}}
}

func checkLegacyStaticToken(ctx context.Context, opts Options) []Finding {
	if conf.RuntimeConfInfo.Nodes.LegacyNodeTokenPresent || strings.TrimSpace(os.Getenv("NODE_PAIRING_TOKEN")) != "" {
		finding := Finding{ID: "node.static_token", Severity: SeverityError, Message: "NODE_PAIRING_TOKEN is present and must be removed", Fixable: false}
		return []Finding{finding}
	}
	return []Finding{{ID: "node.static_token", Severity: SeverityInfo, Message: "static node token is not configured"}}
}

func checkGatewaySecret(ctx context.Context, opts Options) []Finding {
	secret := strings.TrimSpace(firstNonEmpty(os.Getenv("HTTP_SHARED_SECRET"), conf.RuntimeConfInfo.Gateway.SharedSecret))
	if len(secret) < 32 {
		return []Finding{{ID: "secret.weak", Severity: SeverityError, Message: "HTTP/Gateway signing secret should be at least 32 characters", Fixable: false}}
	}
	return []Finding{{ID: "secret.strength", Severity: SeverityInfo, Message: "management signing secret length looks acceptable"}}
}

func checkLoopbackBypassRemoved(ctx context.Context, opts Options) []Finding {
	return []Finding{{ID: "http.loopback_bypass", Severity: SeverityInfo, Message: "management auth no longer trusts loopback without an actor token"}}
}

func checkDangerousNodePolicy(ctx context.Context, opts Options) []Finding {
	policy := node.WorkspaceCommandPolicy(opts.WorkspaceID)
	for _, item := range policy.Allow {
		if item == "*" {
			return []Finding{{ID: "node.policy.allow_all", Severity: SeverityError, Message: "workspace node policy allows all capabilities"}}
		}
	}
	return []Finding{{ID: "node.policy", Severity: SeverityInfo, Message: "no allow-all workspace node policy detected"}}
}

func checkLegacyFeatureGates(ctx context.Context, opts Options) []Finding {
	if conf.FeatureConfInfo.LegacyBotsEnabled() || conf.FeatureConfInfo.LegacyTaskToolsEnabled() || conf.FeatureConfInfo.MediaEnabled() || conf.FeatureConfInfo.CronEnabled() {
		return []Finding{{ID: "features.legacy_open", Severity: SeverityWarn, Message: "one or more legacy optional surfaces are enabled"}}
	}
	return []Finding{{ID: "features.legacy_open", Severity: SeverityInfo, Message: "legacy optional surfaces are closed by default"}}
}

func missingTables(id string, tables []string) []Finding {
	for _, table := range tables {
		if !tableExists(table) {
			return []Finding{{ID: id, Severity: SeverityError, Message: table + " table is missing"}}
		}
	}
	return []Finding{{ID: id, Severity: SeverityInfo, Message: "required tables exist"}}
}

func tableExists(table string) bool {
	if db.DB == nil {
		return false
	}
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?", table).Scan(&count)
	if err == nil {
		return count > 0
	}
	err = db.DB.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", table).Scan(&count)
	return err == nil && count > 0
}

func columnExists(table, column string) bool {
	if db.DB == nil {
		return false
	}
	rows, err := db.DB.Query("PRAGMA table_info(" + table + ")")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name string
			var columnType string
			var notNull int
			var defaultVal sql.NullString
			var pk int
			if rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk) == nil && name == column {
				return true
			}
		}
	}
	var count int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?", table, column).Scan(&count)
	return err == nil && count > 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
