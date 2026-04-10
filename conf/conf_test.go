package conf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitConf_AllEnvVars(t *testing.T) {
	// 准备环境变量
	os.Setenv("TELEGRAM_BOT_TOKEN", "test_bot_token")
	os.Setenv("DEEPSEEK_TOKEN", "test_deepseek_token")
	os.Setenv("CUSTOM_URL", "https://example.com")
	os.Setenv("TYPE", "pro")
	os.Setenv("VOLC_AK", "volc-ak")
	os.Setenv("VOLC_SK", "volc-sk")
	os.Setenv("DB_TYPE", "mysql")
	os.Setenv("DB_CONF", "user:pass@tcp(127.0.0.1:3306)/dbname")
	os.Setenv("ALLOWED_USER_IDS", "1001,1002")
	os.Setenv("ALLOWED_GROUP_IDS", "-2001,-2002")
	os.Setenv("LLM_PROXY", "http://proxy.deepseek")
	os.Setenv("ROBOT_PROXY", "http://proxy.telegram")
	os.Setenv("LANG", "zh-CN")
	os.Setenv("TOKEN_PER_USER", "888")
	os.Setenv("ADMIN_USER_IDS", "9999,8888")
	os.Setenv("NEED_AT_BOT", "true")
	os.Setenv("MAX_USER_CHAT", "10")
	os.Setenv("VIDEO_TOKEN", "video_token_abc")
	os.Setenv("HTTP_HOST", "8888")
	os.Setenv("USE_TOOLS", "false")
	os.Setenv("OPENAI_TOKEN", "openai_test")
	os.Setenv("GEMINI_TOKEN", "gemini_test")
	os.Setenv("ERNIE_AK", "ernie-ak")
	os.Setenv("ERNIE_SK", "ernie-sk")

	os.Setenv("VOL_AUDIO_APP_ID", "test-audio-app-id")
	os.Setenv("VOL_AUDIO_TOKEN", "test-audio-token")
	os.Setenv("VOL_AUDIO_REC_CLUSTER", "test-cluster")

	os.Setenv("TELEGRAM_BOT_TOKEN", "test_bot_token")
	os.Setenv("DEEPSEEK_TOKEN", "test_deepseek_token")
	os.Setenv("FREQUENCY_PENALTY", "0.5")
	os.Setenv("MAX_TOKENS", "2048")
	os.Setenv("PRESENCE_PENALTY", "1.0")
	os.Setenv("TEMPERATURE", "0.9")
	os.Setenv("TOP_P", "0.8")
	os.Setenv("STOP", "stop-sequence")
	os.Setenv("LOG_PROBS", "true")
	os.Setenv("TOP_LOG_PROBS", "5")

	os.Setenv("TELEGRAM_BOT_TOKEN", "test_bot_token")
	os.Setenv("DEEPSEEK_TOKEN", "test_deepseek_token")
	os.Setenv("REQ_KEY", "test-req-key")
	os.Setenv("MODEL_VERSION", "v2.1")
	os.Setenv("REQ_SCHEDULE_CONF", "scheduleA")
	os.Setenv("SEED", "1234")
	os.Setenv("SCALE", "2.5")
	os.Setenv("DDIM_Steps", "30")
	os.Setenv("WIDTH", "512")
	os.Setenv("HEIGHT", "768")
	os.Setenv("USE_PRE_LLM", "true")
	os.Setenv("USE_SR", "false")
	os.Setenv("RETURN_URL", "true")
	os.Setenv("ADD_LOGO", "false")
	os.Setenv("POSITION", "bottom-right")
	os.Setenv("PHOTO_LANGUAGE", "1")
	os.Setenv("OPACITY", "0.75")
	os.Setenv("LOGO_TEXT_CONTENT", "Test Logo")

	os.Setenv("TELEGRAM_BOT_TOKEN", "test_bot_token")
	os.Setenv("DEEPSEEK_TOKEN", "test_deepseek_token")
	os.Setenv("EMBEDDING_TYPE", "huggingface")
	os.Setenv("EMBEDDING_BASE_URL", "http://hf-embeddings:80")
	os.Setenv("EMBEDDING_MODEL_ID", "BAAI/bge-small-zh-v1.5")
	os.Setenv("EMBEDDING_QUERY_INSTRUCTION", "为这个句子生成表示以用于检索相关文章：")
	os.Setenv("KNOWLEDGE_PATH", "/data/knowledge")
	os.Setenv("CHUNK_SIZE", "500")
	os.Setenv("CHUNK_OVERLAP", "50")
	os.Setenv("RERANKER_TOP_N", "6")
	os.Setenv("POSTGRES_DSN", "postgres://tinyclaw:tinyclawpass@postgres:5432/tinyclaw?sslmode=disable")
	os.Setenv("REDIS_ADDR", "redis:6379")
	os.Setenv("REDIS_PASSWORD", "redis-pass")
	os.Setenv("REDIS_DB", "2")
	os.Setenv("MINIO_ENDPOINT", "minio:9000")
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin-secret")
	os.Setenv("MINIO_BUCKET", "tinyclaw-knowledge")
	os.Setenv("MINIO_USE_SSL", "false")
	os.Setenv("DEFAULT_KNOWLEDGE_BASE", "default-kb")
	os.Setenv("DEFAULT_COLLECTION", "team-docs")
	os.Setenv("KNOWLEDGE_AUTO_MIGRATE", "true")
	os.Setenv("RERANKER_BASE_URL", "http://reranker:8081")
	os.Setenv("DENSE_SCORE_THRESHOLD", "0.66")
	os.Setenv("LEXICAL_SCORE_THRESHOLD", "0.07")
	os.Setenv("FUSED_SCORE_THRESHOLD", "0.03")
	os.Setenv("RERANKER_SCORE_THRESHOLD", "0.22")
	os.Setenv("ENABLE_KNOWLEDGE", "true")
	os.Setenv("ENABLE_MEDIA", "true")
	os.Setenv("ENABLE_CRON", "true")
	os.Setenv("ENABLE_LEGACY_BOTS", "true")
	os.Setenv("ENABLE_LEGACY_MCP_PROXY", "true")
	os.Setenv("ENABLE_LEGACY_TASK_TOOLS", "true")
	os.Setenv("ENABLE_EXPERIMENTAL_WORKFLOW", "true")

	os.Setenv("MCP_CONF_PATH", "./conf/mcp/mcp.json")

	os.Setenv("TELEGRAM_BOT_TOKEN", "test_bot_token")
	os.Setenv("DEEPSEEK_TOKEN", "test_deepseek_token")
	os.Setenv("VOL_VIDEO_MODEL", "model-v1")
	os.Setenv("RADIO", "radio-123")
	os.Setenv("DURATION", "120")
	os.Setenv("FPS", "30")
	os.Setenv("RESOLUTION", "1920x1080")
	os.Setenv("WATERMARK", "true")

	// 调用初始化函数
	InitConf()

	// 断言检查
	assertEqual(t, BaseConfInfo.TelegramBotToken, "test_bot_token", "BotToken")
	assertEqual(t, BaseConfInfo.DeepseekToken, "test_deepseek_token", "DeepseekToken")
	assertEqual(t, BaseConfInfo.CustomUrl, "https://example.com", "CustomUrl")
	assertEqual(t, BaseConfInfo.Type, "pro", "Type")
	assertEqual(t, BaseConfInfo.VolcAK, "volc-ak", "VolcAK")
	assertEqual(t, BaseConfInfo.VolcSK, "volc-sk", "VolcSK")
	assertEqual(t, BaseConfInfo.DBType, "mysql", "DBType")
	assertEqual(t, BaseConfInfo.DBConf, "user:pass@tcp(127.0.0.1:3306)/dbname", "DBConf")
	assertEqual(t, BaseConfInfo.LLMProxy, "http://proxy.deepseek", "LLMProxy")
	assertEqual(t, BaseConfInfo.RobotProxy, "http://proxy.telegram", "RobotProxy")
	assertEqual(t, BaseConfInfo.Lang, "zh-CN", "Lang")
	assertMapContains(t, BaseConfInfo.AllowedUserIds, "1001", "AllowedUserIds[1001]")
	assertMapContains(t, BaseConfInfo.AllowedUserIds, "1002", "AllowedUserIds[1002]")
	assertMapContains(t, BaseConfInfo.AllowedGroupIds, "-2001", "AllowedGroupIds[-2001]")
	assertMapContains(t, BaseConfInfo.AllowedGroupIds, "-2002", "AllowedGroupIds[-2002]")
	assertMapContains(t, BaseConfInfo.PrivilegedUserIds, "9999", "PrivilegedUserIds[9999]")
	assertMapContains(t, BaseConfInfo.PrivilegedUserIds, "8888", "PrivilegedUserIds[8888]")
	assertInt(t, BaseConfInfo.TokenPerUser, 888, "TokenPerUser")
	assertInt(t, BaseConfInfo.MaxUserChat, 10, "MaxUserChat")
	assertEqual(t, BaseConfInfo.HTTPHost, "8888", "HTTPPort")
	assertBool(t, BaseConfInfo.UseTools, false, "UseTools")
	assertEqual(t, BaseConfInfo.OpenAIToken, "openai_test", "OpenAIToken")
	assertEqual(t, BaseConfInfo.GeminiToken, "gemini_test", "GeminiToken")
	assertEqual(t, BaseConfInfo.ErnieAK, "ernie-ak", "ErnieAK")
	assertEqual(t, BaseConfInfo.ErnieSK, "ernie-sk", "ErnieSK")

	assertEqual(t, AudioConfInfo.VolAudioAppID, "test-audio-app-id", "AudioAppID")
	assertEqual(t, AudioConfInfo.VolAudioToken, "test-audio-token", "AudioToken")
	assertEqual(t, AudioConfInfo.VolAudioRecCluster, "test-cluster", "AudioCluster")

	assertFloatEqual(t, LLMConfInfo.FrequencyPenalty, 0.5, "FrequencyPenalty")
	assertInt(t, LLMConfInfo.MaxTokens, 2048, "MaxTokens")
	assertFloatEqual(t, LLMConfInfo.PresencePenalty, 1.0, "PresencePenalty")
	assertFloatEqual(t, LLMConfInfo.Temperature, 0.9, "Temperature")
	assertFloatEqual(t, LLMConfInfo.TopP, 0.8, "TopP")
	assertBool(t, LLMConfInfo.LogProbs, true, "LogProbs")
	assertInt(t, LLMConfInfo.TopLogProbs, 5, "TopLogProbs")

	assertEqual(t, PhotoConfInfo.ReqKey, "test-req-key", "ReqKey")
	assertEqual(t, PhotoConfInfo.ModelVersion, "v2.1", "ModelVersion")
	assertEqual(t, PhotoConfInfo.ReqScheduleConf, "scheduleA", "ReqScheduleConf")
	assertInt(t, PhotoConfInfo.Seed, 1234, "Seed")
	assertFloatEqual(t, PhotoConfInfo.Scale, 2.5, "Scale")
	assertInt(t, PhotoConfInfo.DDIMSteps, 30, "DDIMSteps")
	assertInt(t, PhotoConfInfo.Width, 512, "Width")
	assertInt(t, PhotoConfInfo.Height, 768, "Height")
	assertBool(t, PhotoConfInfo.UsePreLLM, true, "UsePreLLM")
	assertBool(t, PhotoConfInfo.UseSr, false, "UseSr")
	assertBool(t, PhotoConfInfo.ReturnUrl, true, "ReturnUrl")
	assertBool(t, PhotoConfInfo.AddLogo, false, "AddLogo")
	assertEqual(t, PhotoConfInfo.Position, "bottom-right", "Position")
	assertInt(t, PhotoConfInfo.Language, 1, "Language")
	assertFloatEqual(t, PhotoConfInfo.Opacity, 0.75, "Opacity")
	assertEqual(t, PhotoConfInfo.LogoTextContent, "Test Logo", "LogoTextContent")

	assertEqual(t, KnowledgeConfInfo.EmbeddingType, "huggingface", "EmbeddingType")
	assertEqual(t, KnowledgeConfInfo.EmbeddingBaseURL, "http://hf-embeddings:80", "EmbeddingBaseURL")
	assertEqual(t, KnowledgeConfInfo.EmbeddingModelID, "BAAI/bge-small-zh-v1.5", "EmbeddingModelID")
	assertEqual(t, KnowledgeConfInfo.EmbeddingQueryInstruction, "为这个句子生成表示以用于检索相关文章：", "EmbeddingQueryInstruction")
	assertEqual(t, KnowledgeConfInfo.KnowledgePath, "/data/knowledge", "KnowledgePath")
	assertInt(t, KnowledgeConfInfo.ChunkSize, 500, "ChunkSize")
	assertInt(t, KnowledgeConfInfo.ChunkOverlap, 50, "ChunkOverlap")
	assertInt(t, KnowledgeConfInfo.RerankerTopN, 6, "RerankerTopN")
	assertEqual(t, KnowledgeConfInfo.PostgresDSN, "postgres://tinyclaw:tinyclawpass@postgres:5432/tinyclaw?sslmode=disable", "PostgresDSN")
	assertEqual(t, KnowledgeConfInfo.RedisAddr, "redis:6379", "RedisAddr")
	assertEqual(t, KnowledgeConfInfo.RedisPassword, "redis-pass", "RedisPassword")
	assertInt(t, KnowledgeConfInfo.RedisDB, 2, "RedisDB")
	assertEqual(t, KnowledgeConfInfo.MinIOEndpoint, "minio:9000", "MinIOEndpoint")
	assertEqual(t, KnowledgeConfInfo.MinIOAccessKey, "minioadmin", "MinIOAccessKey")
	assertEqual(t, KnowledgeConfInfo.MinIOSecretKey, "minioadmin-secret", "MinIOSecretKey")
	assertEqual(t, KnowledgeConfInfo.MinIOBucket, "tinyclaw-knowledge", "MinIOBucket")
	assertBool(t, KnowledgeConfInfo.MinIOUseSSL, false, "MinIOUseSSL")
	assertEqual(t, KnowledgeConfInfo.DefaultKnowledgeBase, "default-kb", "DefaultKnowledgeBase")
	assertEqual(t, KnowledgeConfInfo.DefaultCollection, "team-docs", "DefaultCollection")
	assertBool(t, KnowledgeConfInfo.KnowledgeAutoMigrate, true, "KnowledgeAutoMigrate")
	assertEqual(t, KnowledgeConfInfo.RerankerBaseURL, "http://reranker:8081", "RerankerBaseURL")
	assertFloatEqual(t, KnowledgeConfInfo.DenseScoreThreshold, 0.66, "DenseScoreThreshold")
	assertFloatEqual(t, KnowledgeConfInfo.LexicalScoreThreshold, 0.07, "LexicalScoreThreshold")
	assertFloatEqual(t, KnowledgeConfInfo.FusedScoreThreshold, 0.03, "FusedScoreThreshold")
	assertFloatEqual(t, KnowledgeConfInfo.RerankerScoreThreshold, 0.22, "RerankerScoreThreshold")
	assertBool(t, KnowledgeConfInfo.FeatureStoreEnabled(), true, "FeatureStoreEnabled")
	assertBool(t, KnowledgeConfInfo.ObjectStorageEnabled(), true, "ObjectStorageEnabled")
	assertBool(t, KnowledgeConfInfo.QueueEnabled(), true, "QueueEnabled")
	assertBool(t, KnowledgeConfInfo.Enabled(), true, "KnowledgeEnabled")
	assertEqual(t, KnowledgeConfInfo.KnowledgeBaseName(), "default-kb", "KnowledgeBaseName")
	assertEqual(t, KnowledgeConfInfo.CollectionName(), "team-docs", "CollectionName")
	assertBool(t, FeatureConfInfo.KnowledgeEnabled(), true, "FeatureKnowledge")
	assertBool(t, FeatureConfInfo.MediaEnabled(), true, "FeatureMedia")
	assertBool(t, FeatureConfInfo.CronEnabled(), true, "FeatureCron")
	assertBool(t, FeatureConfInfo.LegacyBotsEnabled(), true, "FeatureLegacyBots")
	assertBool(t, FeatureConfInfo.LegacyMCPProxyEnabled(), true, "FeatureLegacyMCPProxy")
	assertBool(t, FeatureConfInfo.LegacyTaskToolsEnabled(), true, "FeatureLegacyTaskTools")
	assertBool(t, FeatureConfInfo.WorkflowEnabled(), true, "FeatureWorkflow")

	assertEqual(t, *ToolsConfInfo.McpConfPath, "./conf/mcp/mcp.json", "MCP_CONF_PATH")

	assertEqual(t, VideoConfInfo.VolVideoModel, "model-v1", "VOL_VIDEO_MODEL")
	assertEqual(t, VideoConfInfo.Radio, "radio-123", "RADIO")
	assertInt(t, VideoConfInfo.Duration, 120, "DURATION")
	assertInt(t, VideoConfInfo.FPS, 30, "FPS")
	assertEqual(t, VideoConfInfo.Resolution, "1920x1080", "RESOLUTION")
	assertBool(t, VideoConfInfo.Watermark, true, "WATERMARK")

	os.Clearenv()
}

func TestGetAbsPathHonorsTinyClawRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TINYCLAW_ROOT", root)

	got := GetAbsPath("data/tinyclaw.db")
	want := root + "/data/tinyclaw.db"
	if got != want {
		t.Fatalf("expected root override path %q, got %q", want, got)
	}
}

func TestNormalizeProjectManagedPathRelocatesStaleAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TINYCLAW_ROOT", root)

	got := normalizeProjectManagedPath("/tmp/legacy/TinyClaw/data/tiny_claw.db", "data")
	want := root + "/data/tiny_claw.db"
	if got != want {
		t.Fatalf("expected stale project path to relocate to %q, got %q", want, got)
	}

	unchanged := normalizeProjectManagedPath("/var/lib/custom/app.db", "data")
	if unchanged != "/var/lib/custom/app.db" {
		t.Fatalf("expected unrelated absolute path to remain unchanged, got %q", unchanged)
	}
}

func TestInitConf_LoadedSnapshotStillAppliesEnvOverrides(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TINYCLAW_ROOT", root)

	origArgs := os.Args
	os.Args = []string{"tinyclaw"}
	defer func() {
		os.Args = origArgs
	}()

	BaseConfInfo = &BaseConf{
		BotName:           "TinyClaw",
		HTTPHost:          ":36060",
		AllowedUserIds:    map[string]bool{},
		AllowedGroupIds:   map[string]bool{},
		PrivilegedUserIds: map[string]bool{},
	}
	AudioConfInfo = new(AudioConf)
	LLMConfInfo = new(LLMConf)
	PhotoConfInfo = new(PhotoConf)
	FeatureConfInfo = new(FeatureConf)
	KnowledgeConfInfo = new(KnowledgeConf)
	VideoConfInfo = new(VideoConf)
	RegisterConfInfo = new(RegisterConf)
	mcpConfPath := filepath.Join(root, "conf", "mcp", "mcp.json")
	ToolsConfInfo = &ToolsConf{McpConfPath: &mcpConfPath}
	AllConf = make(map[string]interface{})
	SaveConf()

	BaseConfInfo = new(BaseConf)
	AudioConfInfo = new(AudioConf)
	LLMConfInfo = new(LLMConf)
	PhotoConfInfo = new(PhotoConf)
	FeatureConfInfo = new(FeatureConf)
	KnowledgeConfInfo = new(KnowledgeConf)
	VideoConfInfo = new(VideoConf)
	RegisterConfInfo = new(RegisterConf)
	ToolsConfInfo = new(ToolsConf)
	AllConf = make(map[string]interface{})

	t.Setenv("TYPE", "aliyun")
	t.Setenv("ALIYUN_TOKEN", "aliyun-test-token")
	t.Setenv("DEFAULT_MODEL", "qwen3.6-plus")
	t.Setenv("USE_TOOLS", "true")
	t.Setenv("PRIVILEGED_USER_IDS", "admin-1,admin-2")

	InitConf()

	assertEqual(t, BaseConfInfo.Type, "aliyun", "Type")
	assertEqual(t, BaseConfInfo.AliyunToken, "aliyun-test-token", "AliyunToken")
	assertEqual(t, BaseConfInfo.DefaultModel, "qwen3.6-plus", "DefaultModel")
	assertBool(t, BaseConfInfo.UseTools, true, "UseTools")
	assertMapContains(t, BaseConfInfo.PrivilegedUserIds, "admin-1", "PrivilegedUserIds[admin-1]")
	assertMapContains(t, BaseConfInfo.PrivilegedUserIds, "admin-2", "PrivilegedUserIds[admin-2]")
}

// 辅助函数
func assertEqual(t *testing.T, got, expected, field string) {
	if got != expected {
		t.Errorf("%s expected '%s', got '%s'", field, expected, got)
	}
}

func assertInt(t *testing.T, got int, expected int, field string) {
	if got != expected {
		t.Errorf("%s expected %d, got %d", field, expected, got)
	}
}

func assertBool(t *testing.T, got bool, expected bool, field string) {
	if got != expected {
		t.Errorf("%s expected %v, got %v", field, expected, got)
	}
}

func assertMapContains(t *testing.T, got map[string]bool, key string, field string) {
	if !got[key] {
		t.Errorf("%s expected key %q to be present, got %#v", field, key, got)
	}
}

func assertFloatEqual(t *testing.T, got, expected float64, field string) {
	if got != expected {
		t.Errorf("%s expected %.2f, got %.2f", field, expected, got)
	}
}
