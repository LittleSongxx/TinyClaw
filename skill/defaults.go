package skill

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/logger"
)

type ValidationReport struct {
	Roots        []string `json:"roots"`
	Skills       []*Skill `json:"skills"`
	Warnings     []string `json:"warnings,omitempty"`
	Total        int      `json:"total"`
	LocalCount   int      `json:"local_count"`
	BuiltinCount int      `json:"builtin_count"`
	LegacyCount  int      `json:"legacy_count"`
}

func DefaultLoadOptions() LoadOptions {
	mcpConfPath := ""
	if conf.ToolsConfInfo != nil && conf.ToolsConfInfo.McpConfPath != nil {
		mcpConfPath = *conf.ToolsConfInfo.McpConfPath
	}

	return LoadOptions{
		SkillRoots:  DefaultRoots(),
		MCPConfPath: mcpConfPath,
	}
}

func LoadDefaultCatalog() (*Catalog, error) {
	return LoadCatalog(DefaultLoadOptions())
}

func Validate(opts LoadOptions) (*ValidationReport, error) {
	catalog, err := LoadCatalog(opts)
	if err != nil {
		return nil, err
	}

	report := &ValidationReport{
		Roots:    append([]string(nil), opts.SkillRoots...),
		Skills:   catalog.List(),
		Warnings: append([]string(nil), catalog.Warnings...),
	}
	if len(report.Roots) == 0 {
		report.Roots = DefaultRoots()
	}

	report.Total = len(report.Skills)
	for _, item := range report.Skills {
		switch item.Source {
		case "local":
			report.LocalCount++
		case "builtin":
			report.BuiltinCount++
		case "legacy":
			report.LegacyCount++
		}
	}

	return report, nil
}

func ValidateDefaultCatalog() (*ValidationReport, error) {
	return Validate(DefaultLoadOptions())
}

func LogDefaultCatalog(ctx context.Context) {
	report, err := ValidateDefaultCatalog()
	if err != nil {
		logger.WarnCtx(ctx, "skill catalog validation failed", "err", err)
		return
	}

	logger.InfoCtx(ctx, "skill catalog loaded",
		"total", report.Total,
		"local", report.LocalCount,
		"builtin", report.BuiltinCount,
		"legacy", report.LegacyCount,
	)
	for _, warning := range report.Warnings {
		logger.WarnCtx(ctx, "skill catalog warning", "warning", warning)
	}
}

func FormatCatalogList(catalog *Catalog) string {
	if catalog == nil {
		return "No skills available."
	}

	items := catalog.List()
	if len(items) == 0 {
		return "No skills available."
	}

	lines := make([]string, 0, len(items)+2)
	lines = append(lines, "Available skills:")
	for _, item := range items {
		if item == nil {
			continue
		}

		source := item.Source
		if source == "" {
			source = "unknown"
		}
		alias := skillAlias(item)
		line := fmt.Sprintf("- %s [%s]", item.Manifest.ID, source)
		if alias != "" {
			line += fmt.Sprintf(" (alias: %s)", alias)
		}
		line += fmt.Sprintf(": %s", item.Manifest.Description)
		if len(item.Manifest.Modes) > 0 {
			line += fmt.Sprintf(" (modes: %s)", strings.Join(item.Manifest.Modes, ", "))
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func FormatMCPList(catalog *Catalog) string {
	if catalog == nil {
		return "No MCP servers available."
	}

	servers := catalog.MCPServers()
	if len(servers) == 0 {
		return "No MCP servers available."
	}

	lines := make([]string, 0, len(servers)*2+1)
	lines = append(lines, "Available MCP servers:")
	for _, server := range servers {
		description := strings.TrimSpace(server.Description)
		if description == "" {
			description = "No description."
		}
		line := fmt.Sprintf("- %s: %s", server.Name, description)
		if server.ToolCount > 0 {
			line += fmt.Sprintf(" (tools: %s)", strings.Join(server.Tools, ", "))
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func FormatValidationReport(report *ValidationReport) string {
	if report == nil {
		return "Skill validation report is unavailable."
	}

	lines := []string{
		fmt.Sprintf("Skill validation summary: total=%d local=%d builtin=%d legacy=%d",
			report.Total, report.LocalCount, report.BuiltinCount, report.LegacyCount),
	}

	if len(report.Roots) > 0 {
		roots := append([]string(nil), report.Roots...)
		sort.Strings(roots)
		lines = append(lines, "Roots: "+strings.Join(roots, ", "))
	}

	if len(report.Warnings) == 0 {
		lines = append(lines, "No validation warnings.")
		return strings.Join(lines, "\n")
	}

	lines = append(lines, "Warnings:")
	for _, warning := range report.Warnings {
		lines = append(lines, "- "+warning)
	}
	return strings.Join(lines, "\n")
}
