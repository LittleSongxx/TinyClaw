package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/i18n"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/metrics"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/session"
	"github.com/LittleSongxx/TinyClaw/tooling"
	"github.com/LittleSongxx/TinyClaw/utils"
	"github.com/LittleSongxx/mcp-client-go/clients"
	godeepseek "github.com/cohesion-org/deepseek-go"
	"github.com/revrost/go-openrouter"
	"github.com/sashabaranov/go-openai"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"google.golang.org/genai"
)

const (
	OneMsgLen       = 3896
	FirstSendLen    = 30
	NonFirstSendLen = 500
	MostLoop        = 15
)

var (
	ToolsJsonErr = errors.New("tools json error")
)

type LLM struct {
	MessageChan chan *param.MsgInfo
	HTTPMsgChan chan string
	Content     string
	Images      [][]byte

	Model string
	Cs    *param.ContextState

	ChatId           string
	UserId           string
	MsgId            string
	PerMsgLen        int
	ContentParameter map[string]string

	LLMClient LLMClient

	Ctx context.Context

	DeepseekTools   []godeepseek.Tool
	VolTools        []*model.Tool
	OpenAITools     []openai.Tool
	GeminiTools     []*genai.Tool
	OpenRouterTools []openrouter.Tool

	WholeContent string // whole answer from llm
	LoopNum      int

	ToolObserver func(tooling.Observation)
	AllowedTools map[string]bool
	ToolBroker   *tooling.Broker

	RuntimeToolGuidance []string
	runtimeToolsReady   bool
}

type LLMClient interface {
	Send(ctx context.Context, l *LLM) error

	GetMessage(role, msg string)

	GetImageMessage(image [][]byte, msg string)

	GetAudioMessage(audio []byte, msg string)

	AppendMessages(client LLMClient)

	SyncSend(ctx context.Context, l *LLM) (string, error)

	GetModel(l *LLM)
}

func (l *LLM) CallLLM() error {
	l.ensureRuntimeTools()
	totalContent := l.GetContent(l.Content)
	l.GetMessages(l.UserId, totalContent)
	l.InsertCharacter(l.Ctx)
	l.LLMClient.GetModel(l)

	logger.InfoCtx(l.Ctx, "msg receive", "userID", l.UserId, "prompt", totalContent, "type",
		utils.GetTxtType(db.GetCtxUserInfo(l.Ctx).LLMConfigRaw), "model", l.Model)

	metrics.APIRequestCount.WithLabelValues(l.Model).Inc()

	var err error
	if conf.BaseConfInfo.IsStreaming {
		err = l.LLMClient.Send(l.Ctx, l)
		if err != nil {
			logger.ErrorCtx(l.Ctx, "Error calling LLM API", "err", err)
			return err
		}
	} else {
		content, err := l.LLMClient.SyncSend(l.Ctx, l)
		if err != nil {
			logger.ErrorCtx(l.Ctx, "Error calling LLM API", "err", err)
			return err
		}

		l.MessageChan <- &param.MsgInfo{
			Content: content,
		}
		l.WholeContent = content
	}

	err = l.InsertOrUpdate()
	if err != nil {
		logger.ErrorCtx(l.Ctx, "insert or update record", "err", err)
		return err
	}

	return nil
}

func (l *LLM) GetContent(content string) string {
	return content
}

func (l *LLM) InsertCharacter(ctx context.Context) {
	if conf.BaseConfInfo.Character != "" {
		if l.ContentParameter != nil {
			tmpl, err := template.New("character").Parse(conf.BaseConfInfo.Character)
			if err != nil {
				logger.ErrorCtx(ctx, "parse template fail", "err", err)
				return
			}

			var buf bytes.Buffer
			err = tmpl.Execute(&buf, l.ContentParameter)
			if err != nil {
				logger.ErrorCtx(ctx, "exec template fail", "err", err)
				return
			}

			logger.InfoCtx(ctx, "character", "character", buf.String())
			l.LLMClient.GetMessage(openai.ChatMessageRoleSystem, buf.String())
		}
	}
}

func NewLLM(opts ...Option) *LLM {
	l := new(LLM)
	l.Cs = new(param.ContextState)
	for _, opt := range opts {
		opt(l)
	}

	switch utils.GetTxtType(db.GetCtxUserInfo(l.Ctx).LLMConfigRaw) {
	case param.Ollama:
		l.LLMClient = &OllamaReq{
			ToolCall:           []godeepseek.ToolCall{},
			ToolMessage:        []godeepseek.ChatCompletionMessage{},
			CurrentToolMessage: []godeepseek.ChatCompletionMessage{},
		}
	default:
		l.LLMClient = &OpenAIReq{
			ToolCall:           []openai.ToolCall{},
			ToolMessage:        []openai.ChatCompletionMessage{},
			CurrentToolMessage: []openai.ChatCompletionMessage{},
		}
	}

	return l
}

func (l *LLM) DirectSendMsg(content string, ignoreLen bool) {
	if !ignoreLen && len([]byte(content)) > l.PerMsgLen {
		content = string([]byte(content)[:l.PerMsgLen])
	}

	if l.MessageChan != nil {
		l.MessageChan <- &param.MsgInfo{
			Content:  content,
			Finished: true,
		}
	}

	if l.HTTPMsgChan != nil {
		l.HTTPMsgChan <- content
	}
}

func (l *LLM) SendMsg(msgInfoContent *param.MsgInfo, content string) *param.MsgInfo {
	if l.MessageChan != nil {
		if l.PerMsgLen == 0 {
			l.PerMsgLen = OneMsgLen
		}

		// exceed max one message length
		if len([]byte(msgInfoContent.Content)) > l.PerMsgLen {
			msgInfoContent.Finished = true
			l.MessageChan <- msgInfoContent
			msgInfoContent = &param.MsgInfo{
				SendLen: NonFirstSendLen,
			}
		}

		msgInfoContent.Content += content
		l.WholeContent += content
		if len(msgInfoContent.Content) > msgInfoContent.SendLen {
			l.MessageChan <- msgInfoContent
			msgInfoContent.SendLen += NonFirstSendLen
		}

		return msgInfoContent
	} else {
		l.WholeContent += content
		l.HTTPMsgChan <- content
		return nil
	}
}

func (l *LLM) OverLoop() bool {
	if l.LoopNum >= MostLoop {
		return true
	}
	l.LoopNum++
	return false
}

func (l *LLM) InsertOrUpdate() error {
	if l.Cs.RecordID == 0 {
		db.InsertMsgRecord(l.Ctx, l.UserId, &db.AQ{
			Question:   l.Content,
			Answer:     l.WholeContent,
			Token:      l.Cs.Token,
			CreateTime: time.Now().Unix(),
		}, true)
		_ = session.AppendConversation(l.Ctx, l.Cs.SessionID, l.Content, l.WholeContent, map[string]string{
			"mode":   "llm",
			"source": l.Cs.Source,
		})
		return nil
	}

	db.InsertMsgRecord(l.Ctx, l.UserId, &db.AQ{
		Question:   l.Content,
		Answer:     l.WholeContent,
		CreateTime: time.Now().Unix(),
	}, false)
	err := db.UpdateRecordInfo(&db.Record{
		ID:     l.Cs.RecordID,
		Answer: l.WholeContent,
		Token:  l.Cs.Token,
		UserId: l.UserId,
		Mode:   utils.GetTxtType(db.GetCtxUserInfo(l.Ctx).LLMConfigRaw),
	})
	if err != nil {
		logger.ErrorCtx(l.Ctx, "update record fail", "err", err)
		return err
	}

	_ = session.AppendConversation(l.Ctx, l.Cs.SessionID, l.Content, l.WholeContent, map[string]string{
		"mode":   "llm",
		"source": l.Cs.Source,
	})

	return nil
}

func (l *LLM) GetMessages(userId string, prompt string) {
	if l.Cs != nil && l.Cs.SessionID != "" {
		items, err := session.RecentContext(l.Ctx, l.Cs.SessionID, conf.RuntimeConfInfo.Sessions.ContextWindow)
		if err == nil && len(items) > 0 {
			for i, record := range items {
				logger.InfoCtx(l.Ctx, "session context", "dialog", i, "role", record.Role, "content", record.Content)
				switch record.Role {
				case session.RoleAssistant:
					l.LLMClient.GetMessage(openai.ChatMessageRoleAssistant, record.Content)
				case session.RoleSystem:
					l.LLMClient.GetMessage(openai.ChatMessageRoleSystem, record.Content)
				default:
					l.LLMClient.GetMessage(openai.ChatMessageRoleUser, record.Content)
				}
			}
		}
	} else {
		msgRecords := db.GetMsgRecord(userId)
		if msgRecords != nil && l.Cs.UseRecord {
			aqs := db.FilterByMaxContextFromLatest(msgRecords.AQs, param.DefaultContextToken)
			for i, record := range aqs {
				if record.Question != "" && record.Answer != "" && record.CreateTime > time.Now().Unix()-int64(conf.BaseConfInfo.ContextExpireTime) {
					logger.InfoCtx(l.Ctx, "context content", "dialog", i, "question:", record.Question, "answer:", record.Answer)
					l.LLMClient.GetMessage(openai.ChatMessageRoleUser, record.Question)
					l.LLMClient.GetMessage(openai.ChatMessageRoleAssistant, record.Answer)
				}
			}
		}
	}

	for _, guidance := range l.RuntimeToolGuidance {
		l.LLMClient.GetMessage(openai.ChatMessageRoleSystem, guidance)
	}

	if len(l.Images) > 0 {
		l.LLMClient.GetImageMessage(l.Images, prompt)
	} else {
		l.LLMClient.GetMessage(openai.ChatMessageRoleUser, prompt)
	}

}

type Option func(p *LLM)

func WithModel(model string) Option {
	return func(p *LLM) {
		p.Model = model
	}
}

func WithContent(content string) Option {
	return func(p *LLM) {
		p.Content = content
	}
}

func WithPerMsgLen(perMsgLen int) Option {
	return func(p *LLM) {
		p.PerMsgLen = perMsgLen
	}
}

func WithMessageChan(messageChan chan *param.MsgInfo) Option {
	return func(p *LLM) {
		p.MessageChan = messageChan
	}
}

func WithHTTPMsgChan(messageChan chan string) Option {
	return func(p *LLM) {
		p.HTTPMsgChan = messageChan
	}
}

func WithChatId(chatId string) Option {
	return func(p *LLM) {
		p.ChatId = chatId
	}
}

func WithUserId(userId string) Option {
	return func(p *LLM) {
		p.UserId = userId
	}
}

func WithMsgId(msgId string) Option {
	return func(p *LLM) {
		p.MsgId = msgId
	}
}

func WithCS(cs *param.ContextState) Option {
	return func(p *LLM) {
		p.Cs = cs
	}
}

func WithImages(images [][]byte) Option {
	return func(p *LLM) {
		p.Images = images
	}
}

func WithTaskTools(taskTool *conf.AgentInfo) Option {
	return func(p *LLM) {
		if taskTool == nil {
			p.DeepseekTools = nil
			p.VolTools = nil
			p.OpenAITools = nil
			p.GeminiTools = nil
			p.OpenRouterTools = nil
			return
		}
		p.DeepseekTools = taskTool.DeepseekTool
		p.VolTools = taskTool.VolTool
		p.OpenAITools = taskTool.OpenAITools
		p.GeminiTools = taskTool.GeminiTools
		p.OpenRouterTools = taskTool.OpenRouterTools
	}
}

func WithContext(ctx context.Context) Option {
	return func(p *LLM) {
		p.Ctx = ctx
	}
}

func WithAllowedToolNames(toolNames []string) Option {
	return func(p *LLM) {
		if len(toolNames) == 0 {
			p.AllowedTools = nil
			return
		}

		p.AllowedTools = make(map[string]bool, len(toolNames))
		for _, name := range toolNames {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			p.AllowedTools[name] = true
		}
	}
}

func WithToolObserver(observer func(tooling.Observation)) Option {
	return func(p *LLM) {
		p.ToolObserver = observer
	}
}

func WithToolBroker(toolBroker *tooling.Broker) Option {
	return func(p *LLM) {
		p.ToolBroker = toolBroker
	}
}

func WithContentParameter(contentParameter map[string]string) Option {
	return func(p *LLM) {
		p.ContentParameter = contentParameter
	}
}

func (l *LLM) ExecMcpReq(ctx context.Context, funcName string, property map[string]interface{}) (string, error) {
	if l != nil && len(l.AllowedTools) > 0 && !l.AllowedTools[funcName] {
		err := fmt.Errorf("tool %s is not allowed in the current skill", funcName)
		l.observeTool(funcName, property, "", err)
		return "", err
	}

	if l != nil && l.ToolBroker != nil {
		result, err := l.ToolBroker.Execute(ctx, tooling.ToolInvocation{
			Name:      funcName,
			Arguments: property,
			SessionID: l.sessionID(),
			NodeID:    toolNodeID(property),
		})
		switch {
		case err == nil:
			return l.finalizeToolResult(funcName, property, result.Output)
		case errors.Is(err, tooling.ErrToolProviderNotFound):
		default:
			logger.ErrorCtx(ctx, "execute runtime tool fail", "err", err, "function", funcName, "argument", property)
			l.observeTool(funcName, property, "", err)
			return "", err
		}
	}

	mc, err := clients.GetMCPClientByToolName(funcName)
	if err != nil {
		logger.ErrorCtx(ctx, "get mcp fail", "err", err, "function", funcName, "argument", property)
		l.observeTool(funcName, property, "", err)
		return "", err
	}

	metrics.MCPRequestCount.WithLabelValues(mc.Conf.Name, funcName).Inc()
	startTime := time.Now()

	var toolsData string
	for i := 0; i < conf.BaseConfInfo.LLMRetryTimes; i++ {
		toolsData, err = mc.ExecTools(ctx, funcName, property)
		if err != nil {
			time.Sleep(time.Duration(conf.BaseConfInfo.LLMRetryInterval) * time.Millisecond)
			continue
		}
		break
	}

	if err != nil {
		logger.ErrorCtx(ctx, "get mcp fail", "err", err, "function", funcName, "argument", property)
		l.observeTool(funcName, property, "", err)
		return "", err
	}

	metrics.MCPRequestDuration.WithLabelValues(mc.Conf.Name, funcName).Observe(time.Since(startTime).Seconds())

	logger.InfoCtx(ctx, "get mcp", "function", funcName, "argument", property, "res", toolsData)
	return l.finalizeToolResult(funcName, property, toolsData)
}

func (l *LLM) finalizeToolResult(funcName string, property map[string]interface{}, toolsData string) (string, error) {
	trimmed := strings.TrimSpace(toolsData)

	if approvalPayload, ok := parsePendingApprovalPayload(trimmed); ok {
		approvalID, _ := approvalPayload["approval_id"].(string)
		summary, _ := approvalPayload["summary"].(string)
		msg := i18n.GetMessage("approval_required", map[string]interface{}{
			"approval_id": approvalID,
			"summary":     summary,
		})
		if msg == "" {
			msg = fmt.Sprintf("该设备操作需要确认：%s\n回复 /approve %s 执行，或回复 /reject %s 取消。", summary, approvalID, approvalID)
		}
		l.DirectSendMsg(msg, true)

		placeholder := i18n.GetMessage("approval_waiting", nil)
		if placeholder == "" {
			placeholder = "[等待用户确认设备操作]"
		}
		l.observeTool(funcName, property, placeholder, nil)
		return placeholder, nil
	}

	if mcpResp, meta, ok := parseImageToolPayload(trimmed); ok {
		l.DirectSendMsg("![image](data:"+mcpResp.MimeType+";base64,"+mcpResp.Data+")", true)
		if conf.BaseConfInfo.SendMcpRes {
			l.DirectSendMsg(buildImageToolSummary(funcName, meta), true)
		}

		llmContent := trimmed
		if !conf.BaseConfInfo.SendMcpMediaToLLM {
			llmContent = "[截图已发送给用户]"
		}
		l.observeTool(funcName, property, llmContent, nil)
		return llmContent, nil
	}

	if conf.BaseConfInfo.SendMcpRes {
		l.DirectSendMsg(i18n.GetMessage("send_mcp_info", map[string]interface{}{
			"function_name": funcName,
			"request_args":  property,
			"response":      sanitizeToolResponseForUser(toolsData),
		}), false)
	}

	l.observeTool(funcName, property, toolsData, nil)
	return toolsData, nil
}

func (l *LLM) ensureRuntimeTools() {
	if l == nil || l.runtimeToolsReady || l.ToolBroker == nil {
		return
	}

	ctx := l.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	specs, err := l.ToolBroker.ListExecutable(ctx)
	l.runtimeToolsReady = true
	if err != nil || len(specs) == 0 {
		if err != nil {
			logger.WarnCtx(ctx, "list runtime tools fail", "err", err)
		}
		return
	}

	l.OpenAITools = appendOpenAITools(l.OpenAITools, tooling.ToOpenAITools(specs)...)
	l.DeepseekTools = appendDeepseekTools(l.DeepseekTools, tooling.ToDeepseekTools(specs)...)

	for _, spec := range specs {
		if spec.Category == tooling.CategoryNode {
			l.RuntimeToolGuidance = append(l.RuntimeToolGuidance,
				`你可以使用 node_* 工具直接操作真实配对的 PC 节点（Windows、macOS 或 Linux）。当用户明确要求截图、打开应用、打开网页、读写文件或执行桌面命令时，应优先调用相应 node 工具，而不是只给出文字说明。用户提到“当前窗口”“这个应用”“记事本界面”等窗口语义时，截图优先使用 active_window。涉及按钮、输入框、复选框、菜单项等控件时，优先先用 node_ui_find 或 node_ui_inspect 找到稳定元素，再执行点击或输入；只有在找不到稳定元素时才退回坐标。若用户没有指定 node_id，可以留空让系统自动选择兼容设备。对明显具有破坏性的操作，应先确认。`)
			break
		}
	}
}

func appendOpenAITools(current []openai.Tool, tools ...openai.Tool) []openai.Tool {
	if len(tools) == 0 {
		return current
	}
	seen := make(map[string]bool, len(current))
	for _, tool := range current {
		if tool.Function != nil {
			seen[tool.Function.Name] = true
		}
	}
	for _, tool := range tools {
		if tool.Function == nil || seen[tool.Function.Name] {
			continue
		}
		current = append(current, tool)
		seen[tool.Function.Name] = true
	}
	return current
}

func appendDeepseekTools(current []godeepseek.Tool, tools ...godeepseek.Tool) []godeepseek.Tool {
	if len(tools) == 0 {
		return current
	}
	seen := make(map[string]bool, len(current))
	for _, tool := range current {
		seen[tool.Function.Name] = true
	}
	for _, tool := range tools {
		if seen[tool.Function.Name] {
			continue
		}
		current = append(current, tool)
		seen[tool.Function.Name] = true
	}
	return current
}

func toolNodeID(arguments map[string]interface{}) string {
	if len(arguments) == 0 {
		return ""
	}
	raw, ok := arguments["node_id"]
	if !ok {
		return ""
	}
	value, _ := raw.(string)
	return strings.TrimSpace(value)
}

func (l *LLM) sessionID() string {
	if l == nil || l.Cs == nil {
		return ""
	}
	return l.Cs.SessionID
}

func (l *LLM) observeTool(funcName string, property map[string]interface{}, output string, err error) {
	if l == nil || l.ToolObserver == nil {
		return
	}

	obs := tooling.Observation{
		Function:  funcName,
		Arguments: property,
		Output:    output,
		CreatedAt: time.Now().Unix(),
	}
	if err != nil {
		obs.Error = err.Error()
	}

	l.ToolObserver(obs)
}

func parsePendingApprovalPayload(raw string) (map[string]interface{}, bool) {
	payload, ok := parseToolPayloadMap(raw)
	if !ok {
		return nil, false
	}
	pending, _ := payload["pending_approval"].(bool)
	return payload, pending
}

func parseImageToolPayload(raw string) (*param.MCPResp, map[string]interface{}, bool) {
	payload, ok := parseToolPayloadMap(raw)
	if !ok {
		return nil, nil, false
	}
	itemType, _ := payload["type"].(string)
	if itemType != "image" {
		return nil, nil, false
	}

	content, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, false
	}
	resp := new(param.MCPResp)
	if err := json.Unmarshal(content, resp); err != nil {
		return nil, nil, false
	}
	meta, _ := payload["meta"].(map[string]interface{})
	return resp, meta, resp.Data != ""
}

func parseToolPayloadMap(raw string) (map[string]interface{}, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || (!strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[")) {
		return nil, false
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func buildImageToolSummary(funcName string, meta map[string]interface{}) string {
	width, _ := numberFromAny(meta["width"])
	height, _ := numberFromAny(meta["height"])
	scope, _ := meta["scope"].(string)
	if scope == "" {
		scope = "virtual_desktop"
	}
	windowTitle := ""
	if window, ok := meta["window"].(map[string]interface{}); ok {
		windowTitle, _ = window["title"].(string)
	}
	if width > 0 && height > 0 {
		if windowTitle != "" {
			return fmt.Sprintf("已执行 %s，并将截图发送给用户。范围：%s，窗口：%s，尺寸：%dx%d。", funcName, scope, windowTitle, width, height)
		}
		return fmt.Sprintf("已执行 %s，并将截图发送给用户。范围：%s，尺寸：%dx%d。", funcName, scope, width, height)
	}
	return fmt.Sprintf("已执行 %s，并将截图发送给用户。", funcName)
}

func numberFromAny(raw interface{}) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}

func sanitizeToolResponseForUser(text string) string {
	cleaned := strings.TrimSpace(text)
	if cleaned == "" {
		return cleaned
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?s)\[MCP 返回的 Base64 图像已直接发送给用户\]`),
		regexp.MustCompile(`(?s)\{这里只是一个示例 Base64 图像字符串.*?\}`),
		regexp.MustCompile(`(?m)^\s*\].*$`),
	}
	for _, pattern := range patterns {
		cleaned = pattern.ReplaceAllString(cleaned, "")
	}

	lines := strings.Split(cleaned, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			continue
		case strings.Contains(trimmed, "Base64 图像"),
			strings.Contains(trimmed, "MCP 返回的图像数据"),
			strings.Contains(trimmed, `"mimeType"`),
			strings.Contains(trimmed, `"type":"image"`):
			continue
		default:
			filtered = append(filtered, strings.TrimLeft(trimmed, "] "))
		}
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}
