package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/utils"
	"github.com/LittleSongxx/mcp-client-go/clients"
	mcpParam "github.com/LittleSongxx/mcp-client-go/clients/param"
)

var (
	mcpSecretKeyPattern = regexp.MustCompile(`(?i)(TOKEN|KEY|SECRET|PASSWORD|CLIENT_ID|CLIENT_SECRET|APP_ID|APP_KEY|REFRESH_TOKEN|JWT|PROFILE)`)
	mcpEnvNamePattern   = regexp.MustCompile(`\b[A-Z][A-Z0-9_]{2,}\b`)
)

var mcpSecretPlaceholderPatterns = []*regexp.Regexp{
	regexp.MustCompile(`<YOUR`),
	regexp.MustCompile(`\bYOUR_[A-Z0-9_]+\b`),
	regexp.MustCompile(`\byour-[a-z0-9-]+\b`),
	regexp.MustCompile(`\byour_[a-z0-9_]+\b`),
	regexp.MustCompile(`\byour key\b`),
	regexp.MustCompile(`\bAPIKEY\b`),
	regexp.MustCompile(`FIGMA_API_ACCESS_TOKEN`),
	regexp.MustCompile(`T01234567`),
	regexp.MustCompile(`xoxb-your`),
	regexp.MustCompile(`your-api-key`),
	regexp.MustCompile(`your-key`),
	regexp.MustCompile(`your-github`),
	regexp.MustCompile(`your-jira`),
	regexp.MustCompile(`your-linear`),
	regexp.MustCompile(`your-notion`),
	regexp.MustCompile(`your-google`),
	regexp.MustCompile(`your-twitter`),
	regexp.MustCompile(`YOUR_PHONE_NUMBER`),
	regexp.MustCompile(`2014\.\.\.222`),
	regexp.MustCompile(`MIIE\.\.\.`),
	regexp.MustCompile(`MIIB\.\.\.`),
}

var mcpSetupPlaceholderPatterns = []*regexp.Regexp{
	regexp.MustCompile(`path/to`),
	regexp.MustCompile(`/path/`),
	regexp.MustCompile(`PATH_TO`),
	regexp.MustCompile(`localhost`),
	regexp.MustCompile(`127\.0\.0\.1`),
	regexp.MustCompile(`your-own-server`),
	regexp.MustCompile(`success-page`),
	regexp.MustCompile(`your-function-prefix`),
	regexp.MustCompile(`your-first-function`),
	regexp.MustCompile(`your-second-function`),
	regexp.MustCompile(`your-tag-key`),
	regexp.MustCompile(`your-tag-value`),
}

func InspectMCPConf(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var config *mcpParam.McpClientGoConfig
	var err error

	if r.Method == http.MethodGet {
		config, err = getMCPConf(ctx)
	} else {
		config = new(mcpParam.McpClientGoConfig)
		err = utils.HandleJsonBody(r, config)
	}
	if err != nil {
		logger.ErrorCtx(ctx, "inspect mcp conf error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	utils.Success(ctx, w, r, BuildMCPInspectData(ctx, config))
}

func BuildMCPInspectData(ctx context.Context, config *mcpParam.McpClientGoConfig) *param.MCPInspectData {
	res := &param.MCPInspectData{
		McpServers:   map[string]*mcpParam.MCPConfig{},
		Availability: map[string]*param.MCPAvailability{},
	}

	if config == nil || config.McpServers == nil {
		return res
	}

	for name, mcpConfig := range config.McpServers {
		res.McpServers[name] = mcpConfig
	}

	names := make([]string, 0, len(config.McpServers))
	for name := range config.McpServers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		res.Availability[name] = inspectSingleMCP(ctx, name, config.McpServers[name])
	}

	return res
}

func inspectSingleMCP(ctx context.Context, name string, config *mcpParam.MCPConfig) *param.MCPAvailability {
	if config == nil {
		return &param.MCPAvailability{
			Statuses: []string{"setup"},
			Notes: map[string]string{
				"zh": "当前模板缺少有效配置，补齐后才能继续检查。",
				"en": "This template is missing a valid configuration, so it must be completed before further checks.",
			},
		}
	}

	registered := false
	if _, err := clients.GetMCPClient(name); err == nil {
		registered = true
	}

	if registered {
		return &param.MCPAvailability{
			Statuses:   []string{"ready"},
			Registered: true,
			Notes: map[string]string{
				"zh": "当前服务已经成功注册到 MCP 客户端，可直接使用。",
				"en": "This MCP service is already registered successfully and can be used now.",
			},
		}
	}

	missingSecrets := collectMissingSecrets(config)
	runtimeIssues := collectRuntimeIssues(ctx, config)
	setupIssues := collectSetupIssues(config)

	statuses := make([]string, 0, 4)
	if len(missingSecrets) > 0 {
		statuses = append(statuses, "secret")
	}
	if len(runtimeIssues) > 0 {
		statuses = append(statuses, "runtime")
	}
	if len(setupIssues) > 0 {
		statuses = append(statuses, "setup")
	}
	if len(statuses) == 0 {
		statuses = append(statuses, "ready")
	}

	return &param.MCPAvailability{
		Statuses: statuses,
		Notes: map[string]string{
			"zh": buildMCPAvailabilityNoteZh(statuses, missingSecrets, runtimeIssues, setupIssues),
			"en": buildMCPAvailabilityNoteEn(statuses, missingSecrets, runtimeIssues, setupIssues),
		},
	}
}

func collectMissingSecrets(config *mcpParam.MCPConfig) []string {
	if config == nil {
		return nil
	}

	requiredKeys := map[string]bool{}

	for key := range config.Env {
		if mcpSecretKeyPattern.MatchString(key) {
			requiredKeys[key] = true
		}
	}

	for _, token := range mcpEnvNamePattern.FindAllString(config.Description, -1) {
		if mcpSecretKeyPattern.MatchString(token) {
			requiredKeys[token] = true
		}
	}

	if len(requiredKeys) == 0 {
		return nil
	}

	missing := make([]string, 0)
	for key := range requiredKeys {
		if rawValue, ok := config.Env[key]; ok {
			value := strings.TrimSpace(rawValue)
			if value == "" || hasSecretPlaceholder(value) {
				missing = append(missing, key)
			}
			continue
		}

		if strings.TrimSpace(os.Getenv(key)) == "" {
			missing = append(missing, key)
		}
	}

	sort.Strings(missing)
	return missing
}

func collectRuntimeIssues(ctx context.Context, config *mcpParam.MCPConfig) []string {
	if config == nil {
		return nil
	}

	issues := make([]string, 0)

	if config.Command != "" && !commandExists(config.Command) {
		issues = append(issues, fmt.Sprintf("command %s", config.Command))
	}

	if config.Url != "" && !urlReachable(ctx, config.Url) {
		issues = append(issues, fmt.Sprintf("url %s", config.Url))
	}

	return issues
}

func collectSetupIssues(config *mcpParam.MCPConfig) []string {
	if config == nil {
		return []string{"empty-config"}
	}

	issues := make([]string, 0)

	if hasSetupPlaceholder(config.Command) {
		issues = append(issues, fmt.Sprintf("command=%s", config.Command))
	}

	if hasSetupPlaceholder(config.Url) {
		issues = append(issues, fmt.Sprintf("url=%s", config.Url))
	}

	for _, arg := range config.Args {
		if hasSetupPlaceholder(arg) {
			issues = append(issues, fmt.Sprintf("arg=%s", arg))
		}
	}

	for key, value := range config.Env {
		if hasSetupPlaceholder(key) || hasSetupPlaceholder(value) {
			issues = append(issues, fmt.Sprintf("env=%s", key))
		}
	}

	return issues
}

func hasSecretPlaceholder(value string) bool {
	for _, pattern := range mcpSecretPlaceholderPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}

func hasSetupPlaceholder(value string) bool {
	if value == "" {
		return false
	}
	for _, pattern := range mcpSetupPlaceholderPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}

func commandExists(command string) bool {
	if command == "" {
		return false
	}
	_, err := exec.LookPath(command)
	return err == nil
}

func urlReachable(ctx context.Context, rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}

	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < http.StatusInternalServerError
}

func buildMCPAvailabilityNoteZh(statuses, missingSecrets, runtimeIssues, setupIssues []string) string {
	if len(statuses) == 1 && statuses[0] == "ready" {
		return "当前环境检查通过，可直接添加或启用这个 MCP 服务。"
	}

	parts := make([]string, 0, 3)
	if len(missingSecrets) > 0 {
		parts = append(parts, "缺少环境变量: "+strings.Join(missingSecrets, ", "))
	}
	if len(runtimeIssues) > 0 {
		parts = append(parts, "运行时问题: "+strings.Join(runtimeIssues, ", "))
	}
	if len(setupIssues) > 0 {
		parts = append(parts, "仍有模板配置: "+strings.Join(setupIssues, ", "))
	}

	return strings.Join(parts, "。") + "。"
}

func buildMCPAvailabilityNoteEn(statuses, missingSecrets, runtimeIssues, setupIssues []string) string {
	if len(statuses) == 1 && statuses[0] == "ready" {
		return "The current environment checks passed, so this MCP service can be added or enabled directly."
	}

	parts := make([]string, 0, 3)
	if len(missingSecrets) > 0 {
		parts = append(parts, "Missing environment variables: "+strings.Join(missingSecrets, ", "))
	}
	if len(runtimeIssues) > 0 {
		parts = append(parts, "Runtime issues: "+strings.Join(runtimeIssues, ", "))
	}
	if len(setupIssues) > 0 {
		parts = append(parts, "Template placeholders still present: "+strings.Join(setupIssues, ", "))
	}

	return strings.Join(parts, ". ") + "."
}
