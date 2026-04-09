package runtimecore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/gateway"
	"github.com/LittleSongxx/TinyClaw/knowledge"
	"github.com/LittleSongxx/TinyClaw/llm"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/recall"
	"github.com/LittleSongxx/TinyClaw/skill"
	"github.com/LittleSongxx/TinyClaw/tooling"
	"github.com/LittleSongxx/TinyClaw/utils"
)

type Service struct {
	toolBroker *tooling.Broker
}

func NewService(toolBroker *tooling.Broker) *Service {
	if toolBroker == nil {
		if svc := gateway.DefaultService(); svc != nil {
			toolBroker = svc.ToolBroker()
		}
	}
	return &Service{toolBroker: toolBroker}
}

func DefaultService() *Service {
	return NewService(nil)
}

func (s *Service) Run(req RunRequest) (*RunResult, error) {
	ctx, err := ensureRuntimeContext(req.Ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	req.Ctx = ctx
	if req.Cs == nil {
		req.Cs = &param.ContextState{UseRecord: true}
	}
	if req.PerMsgLen <= 0 {
		req.PerMsgLen = llm.OneMsgLen
	}
	if req.ToolBroker == nil {
		req.ToolBroker = s.toolBroker
	}

	switch req.Mode {
	case ModeChat:
		return s.runChat(req)
	case ModeSkill:
		return s.runStructured(req, ModeSkill)
	case ModeMCP:
		return s.runStructured(req, ModeMCP)
	case ModeWorkflow:
		return s.runStructured(req, ModeTask)
	case ModeTask, "":
		return s.runStructured(req, ModeTask)
	default:
		return nil, fmt.Errorf("unsupported runtime mode: %s", req.Mode)
	}
}

func (s *Service) EffectiveTools(ctx context.Context) (*ToolInventory, error) {
	runtimeTools := make([]tooling.ToolSpec, 0)
	if s != nil && s.toolBroker != nil {
		items, err := s.toolBroker.ListExecutable(ctx)
		if err != nil {
			return nil, err
		}
		runtimeTools = items
	}

	legacyTools := tooling.NewRegistryFromTaskTools().List()
	merged := make([]tooling.ToolSpec, 0, len(runtimeTools)+len(legacyTools))
	seen := make(map[string]bool, len(runtimeTools)+len(legacyTools))
	for _, item := range runtimeTools {
		if seen[item.Name] {
			continue
		}
		merged = append(merged, item)
		seen[item.Name] = true
	}
	for _, item := range legacyTools {
		if seen[item.Name] {
			continue
		}
		merged = append(merged, item)
		seen[item.Name] = true
	}

	return &ToolInventory{
		Count:        len(merged),
		RuntimeCount: len(runtimeTools),
		LegacyCount:  len(legacyTools),
		Tools:        merged,
		RuntimeTools: runtimeTools,
		LegacyTools:  legacyTools,
	}, nil
}

func (s *Service) SkillsStatus(ctx context.Context) (*SkillsStatus, error) {
	catalog, err := skill.LoadDefaultCatalog()
	if err != nil {
		return nil, err
	}

	items := catalog.List()
	out := make([]SkillDescriptor, 0, len(items))
	byMode := make(map[string]int)
	for _, item := range items {
		if item == nil {
			continue
		}
		for _, mode := range item.Manifest.Modes {
			byMode[mode]++
		}
		out = append(out, SkillDescriptor{
			ID:           item.Manifest.ID,
			Name:         item.Manifest.Name,
			Version:      item.Manifest.Version,
			Description:  item.Manifest.Description,
			Modes:        append([]string(nil), item.Manifest.Modes...),
			Triggers:     append([]string(nil), item.Manifest.Triggers...),
			AllowedTools: append([]string(nil), item.Manifest.AllowedTools...),
			Memory:       item.Manifest.Memory,
			Priority:     item.Manifest.Priority,
			Legacy:       item.Legacy,
			Path:         item.Path,
		})
	}

	return &SkillsStatus{
		Count:    len(out),
		Warnings: append([]string(nil), catalog.Warnings...),
		Servers:  catalog.MCPServers(),
		Skills:   out,
		ByMode:   byMode,
	}, nil
}

func (s *Service) KnowledgeStatus(ctx context.Context) recall.KnowledgeStatus {
	status := recall.KnowledgeStatus{
		Enabled:                knowledge.Enabled(),
		Embedder:               conf.KnowledgeConfInfo.EmbeddingType,
		VectorStore:            "pgvector",
		DefaultKnowledgeBase:   conf.KnowledgeConfInfo.KnowledgeBaseName(),
		DefaultCollection:      conf.KnowledgeConfInfo.CollectionName(),
		AsyncIngestion:         conf.KnowledgeConfInfo.QueueEnabled(),
		ObjectStorage:          conf.KnowledgeConfInfo.ObjectStorageEnabled(),
		Queue:                  conf.KnowledgeConfInfo.QueueEnabled(),
		RerankerEnabled:        strings.TrimSpace(conf.KnowledgeConfInfo.RerankerBaseURL) != "",
		RerankerBaseURL:        conf.KnowledgeConfInfo.RerankerBaseURL,
		DenseScoreThreshold:    conf.KnowledgeConfInfo.DenseScoreThreshold,
		LexicalScoreThreshold:  conf.KnowledgeConfInfo.LexicalScoreThreshold,
		FusedScoreThreshold:    conf.KnowledgeConfInfo.FusedScoreThreshold,
		RerankerScoreThreshold: conf.KnowledgeConfInfo.RerankerScoreThreshold,
	}

	if knowledge.Enabled() {
		status.Backend = "postgres+pgvector"
		return status
	}

	status.Backend = "disabled"
	return status
}

func (s *Service) SearchKnowledge(ctx context.Context, query string) ([]recall.RecallHit, error) {
	debug, err := knowledge.DebugRetrieve(ctx, query)
	if err != nil {
		return nil, err
	}

	hits := make([]recall.RecallHit, 0, len(debug.Hits))
	for _, hit := range debug.Hits {
		hits = append(hits, recall.RecallHit{
			ID:       fmt.Sprintf("%d", hit.ChunkID),
			Corpus:   recall.CorpusKnowledge,
			Source:   hit.DocumentName,
			Title:    hit.DocumentName,
			Content:  hit.Content,
			Score:    hit.FinalScore,
			Citation: hit.CitationLabel,
			Metadata: hit.Metadata,
		})
	}
	return hits, nil
}

func (s *Service) runStructured(req RunRequest, mode Mode) (*RunResult, error) {
	taskReq := &llm.LLMTaskReq{
		MessageChan: req.MessageChan,
		HTTPMsgChan: req.HTTPMsgChan,
		Content:     req.Input,
		PerMsgLen:   req.PerMsgLen,
		UserId:      req.UserID,
		ChatId:      req.ChatID,
		MsgId:       req.MsgID,
		ReplayOf:    req.ReplayOf,
		SkillID:     req.SkillID,
		Cs:          req.Cs,
		Ctx:         req.Ctx,
	}

	var (
		run *db.AgentRun
		err error
	)
	switch mode {
	case ModeSkill:
		run, err = taskReq.ExecuteSkillRun()
	case ModeMCP:
		run, err = taskReq.ExecuteMcpRun()
	default:
		run, err = taskReq.ExecuteTaskRun()
	}
	if err != nil {
		return nil, err
	}
	return &RunResult{
		Run:    run,
		Output: run.FinalOutput,
		Mode:   mode,
	}, nil
}

func (s *Service) runChat(req RunRequest) (*RunResult, error) {
	run := &db.AgentRun{
		UserId: req.UserID,
		ChatId: req.ChatID,
		MsgId:  req.MsgID,
		Mode:   string(ModeChat),
		Input:  req.Input,
		Status: "running",
	}

	runID, err := db.InsertAgentRun(run)
	if err != nil {
		logger.ErrorCtx(req.Ctx, "insert chat run fail", "err", err)
	} else {
		run.ID = runID
	}

	step := &db.AgentStep{
		RunID:     run.ID,
		StepIndex: 1,
		Kind:      "executor",
		Name:      "chat",
		Input:     req.Input,
		Status:    "running",
	}
	if run.ID != 0 {
		stepID, stepInsertErr := db.InsertAgentStep(step)
		if stepInsertErr == nil {
			step.ID = stepID
		}
	}

	opts := []llm.Option{
		llm.WithChatId(req.ChatID),
		llm.WithUserId(req.UserID),
		llm.WithMsgId(req.MsgID),
		llm.WithMessageChan(req.MessageChan),
		llm.WithHTTPMsgChan(req.HTTPMsgChan),
		llm.WithContent(req.Input),
		llm.WithPerMsgLen(req.PerMsgLen),
		llm.WithCS(req.Cs),
		llm.WithContext(req.Ctx),
		llm.WithContentParameter(req.ContentParameter),
		llm.WithTaskTools(&conf.AgentInfo{
			DeepseekTool: conf.DeepseekTools,
			VolTool:      conf.VolTools,
			OpenAITools:  conf.OpenAITools,
			GeminiTools:  conf.GeminiTools,
		}),
		llm.WithToolBroker(req.ToolBroker),
		llm.WithImages(req.Images),
	}

	useRecall := determineRecallUsage(req.UseRecall, req.Input)
	provider := utils.GetTxtType(db.GetCtxUserInfo(req.Ctx).LLMConfigRaw)
	output := ""

	if useRecall {
		step.Name = "knowledge_chat"
		step.ToolName = "knowledge_search"
	}

	if step.ID != 0 {
		_ = db.UpdateAgentStep(step)
	}

	if useRecall {
		dpLLM := knowledge.NewRuntime(opts...)
		_, err = dpLLM.Call(req.Ctx, req.Input)
		output = dpLLM.LLM.WholeContent
		step.Model = dpLLM.LLM.Model
		step.Provider = provider
	} else {
		llmClient := llm.NewLLM(opts...)
		err = llmClient.CallLLM()
		output = llmClient.WholeContent
		step.Model = llmClient.Model
		step.Provider = provider
	}

	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		step.Status = "failed"
		step.Error = err.Error()
		step.RawOutput = output
	} else {
		run.Status = "succeeded"
		run.FinalOutput = output
		step.Status = "succeeded"
		step.RawOutput = output
	}

	run.TokenTotal = req.Cs.Token
	run.StepCount = 1
	step.Token = req.Cs.Token

	if step.ID != 0 {
		_ = db.UpdateAgentStep(step)
	}
	if run.ID != 0 {
		run.UpdateTime = time.Now().Unix()
		_ = db.UpdateAgentRun(run)
	}

	return &RunResult{
		Run:        run,
		Output:     output,
		Mode:       ModeChat,
		UsedRecall: useRecall,
	}, err
}

func determineRecallUsage(explicit *bool, input string) bool {
	if explicit != nil {
		return *explicit
	}
	if shouldPreferRuntimeTools(input) {
		return false
	}
	return knowledge.Enabled()
}

func shouldPreferRuntimeTools(input string) bool {
	text := strings.ToLower(strings.TrimSpace(input))
	if text == "" {
		return false
	}
	if strings.HasPrefix(text, "/") || strings.HasPrefix(text, "$") {
		return true
	}

	directPhrases := []string{
		"电脑节点", "pc节点", "节点列表", "在线节点", "可用节点", "当前节点",
		"node list", "list nodes", "list devices", "available nodes", "connected devices", "connected pc",
		"screen snapshot", "take screenshot", "take a screenshot",
		"高德", "amap", "地图坐标", "地理坐标", "经纬度", "map coordinates",
	}
	for _, phrase := range directPhrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}

	verbs := []string{
		"打开", "启动", "列出", "查看", "截图", "截屏", "拍屏", "点击", "输入", "键入", "聚焦", "关闭",
		"执行", "运行", "读取", "读出", "写入", "保存", "打开", "查询", "搜索", "查找", "定位", "导航",
		"open", "launch", "list", "show", "take", "search", "find", "locate", "navigate",
		"click", "type", "focus", "close", "run", "execute", "read", "write",
	}
	targets := []string{
		"节点", "电脑", "设备", "屏幕", "桌面", "窗口", "记事本", "浏览器", "网页", "应用", "程序",
		"文件", "目录", "文件夹", "地图", "位置", "地点", "坐标", "路线", "导航", "高德",
		"node", "device", "screen", "desktop", "window", "browser", "web page",
		"app", "application", "file", "folder", "directory", "map", "location", "coordinates", "route",
	}

	hasVerb := false
	for _, verb := range verbs {
		if strings.Contains(text, verb) {
			hasVerb = true
			break
		}
	}
	if !hasVerb {
		return false
	}

	for _, target := range targets {
		if strings.Contains(text, target) {
			return true
		}
	}
	return false
}

func ensureRuntimeContext(ctx context.Context, userID string) (context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if db.GetCtxUserInfo(ctx) != nil || strings.TrimSpace(userID) == "" {
		return ctx, nil
	}

	userInfo, err := db.GetUserByID(userID)
	if err != nil {
		return ctx, err
	}
	if userInfo == nil || userInfo.ID == 0 {
		if _, err = db.InsertUser(userID, utils.GetDefaultLLMConfig()); err != nil {
			return ctx, err
		}
		userInfo, err = db.GetUserByID(userID)
		if err != nil {
			return ctx, err
		}
	}
	if userInfo == nil {
		return ctx, fmt.Errorf("user %s not found", userID)
	}
	if userInfo.LLMConfigRaw == nil {
		userInfo.LLMConfigRaw = new(param.LLMConfig)
	}
	return context.WithValue(ctx, "user_info", userInfo), nil
}
