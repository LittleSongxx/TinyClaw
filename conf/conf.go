package conf

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/logger"
)

type BaseConf struct {
	StartTime int64 `json:"-"`
	ImageDay  int   `json:"-"`

	TelegramBotToken        string `json:"telegram_bot_token"`
	DiscordBotToken         string `json:"discord_bot_token"`
	SlackBotToken           string `json:"slack_bot_token"`
	SlackAppToken           string `json:"slack_app_token"`
	LarkAPPID               string `json:"lark_app_id"`
	LarkAppSecret           string `json:"lark_app_secret"`
	DingClientId            string `json:"ding_client_id"`
	DingClientSecret        string `json:"ding_client_secret"`
	ComWechatToken          string `json:"com_wechat_token"`
	ComWechatEncodingAESKey string `json:"com_wechat_encoding_aes_key"`
	ComWechatCorpID         string `json:"com_wechat_corp_id"`
	ComWechatSecret         string `json:"com_wechat_secret"`
	ComWechatAgentID        string `json:"com_wechat_agent_id"`
	WechatAppID             string `json:"wechat_app_id"`
	WechatAppSecret         string `json:"wechat_app_secret"`
	WechatToken             string `json:"wechat_token"`
	WechatEncodingAESKey    string `json:"wechat_encoding_aes_key"`
	WechatActive            bool   `json:"wechat_active"`
	QQAppID                 string `json:"qq_app_id"`
	QQAppSecret             string `json:"qq_app_secret"`
	QQOneBotReceiveToken    string `json:"qq_one_bot_receive_token"`
	QQOneBotSendToken       string `json:"qq_one_bot_send_token"`
	QQOneBotHttpServer      string `json:"qq_one_bot_http_server"`

	DeepseekToken     string `json:"deepseek_token"`
	OpenAIToken       string `json:"openai_token"`
	GeminiToken       string `json:"gemini_token"`
	OpenRouterToken   string `json:"open_router_token"`
	AI302Token        string `json:"ai_302_token"`
	VolToken          string `json:"vol_token"`
	AliyunToken       string `json:"aliyun_token"`
	ChatAnyWhereToken string `json:"chat_any_where_token"`
	ErnieAK           string `json:"ernie_ak"`
	ErnieSK           string `json:"ernie_sk"`

	BotName           string `json:"bot_name"`
	Type              string `json:"type"`
	MediaType         string `json:"media_type"`
	CustomUrl         string `json:"custom_url"`
	CustomPath        string `json:"custom_path"`
	VolcAK            string `json:"volc_ak"`
	VolcSK            string `json:"volc_sk"`
	DBType            string `json:"db_type"`
	DBConf            string `json:"db_conf"`
	LLMProxy          string `json:"llm_proxy"`
	RobotProxy        string `json:"robot_proxy"`
	Lang              string `json:"lang"`
	TokenPerUser      int    `json:"token_per_user"`
	MaxUserChat       int    `json:"max_user_chat"`
	HTTPHost          string `json:"http_host"`
	UseTools          bool   `json:"use_tools"`
	MaxQAPair         int    `json:"max_qa_pari"`
	Character         string `json:"character"`
	SmartMode         bool   `json:"smart_mode"`
	ContextExpireTime int    `json:"context_expire_time"`
	Powered           string `json:"powered"`
	SendMcpRes        bool   `json:"send_mcp_res"`
	SendMcpMediaToLLM bool   `json:"send_mcp_media_to_llm"`
	DefaultModel      string `json:"default_model"`
	LLMRetryTimes     int    `json:"llm_retry_times"`
	LLMRetryInterval  int    `json:"llm_retry_interval"`
	LLMOptionParam    bool   `json:"llm_option_param"`
	ImagePath         string `json:"image_path"`
	IsStreaming       bool   `json:"is_streaming"`

	CrtFile string `json:"crt_file"`
	KeyFile string `json:"key_file"`
	CaFile  string `json:"ca_file"`

	AllowedUserIds    map[string]bool `json:"allowed_user_ids"`
	AllowedGroupIds   map[string]bool `json:"allowed_group_ids"`
	PrivilegedUserIds map[string]bool `json:"privileged_user_ids"`
}

var (
	BaseConfInfo = new(BaseConf)
	AllConf      = make(map[string]interface{})
)

func InitConf() {
	BaseConfInfo.StartTime = time.Now().Unix()
	if loadConf() {
		allowedUserIds, allowedGroupIds, privilegedUserIds := applyEnvOverrides("", "", "")
		normalizeLoadedConfPaths()
		logConf(allowedUserIds, allowedGroupIds, privilegedUserIds)
		SaveConf()
		return
	}

	flag.StringVar(&BaseConfInfo.TelegramBotToken, "telegram_bot_token", "", "Telegram bot tokens")
	flag.StringVar(&BaseConfInfo.DiscordBotToken, "discord_bot_token", "", "Discord bot tokens")
	flag.StringVar(&BaseConfInfo.SlackBotToken, "slack_bot_token", "", "Slack bot tokens")
	flag.StringVar(&BaseConfInfo.SlackAppToken, "slack_app_token", "", "Slack app tokens")
	flag.StringVar(&BaseConfInfo.LarkAPPID, "lark_app_id", "", "Lark app id")
	flag.StringVar(&BaseConfInfo.LarkAppSecret, "lark_app_secret", "", "Lark app secret")
	flag.StringVar(&BaseConfInfo.DingClientId, "ding_client_id", "", "Dingding client id")
	flag.StringVar(&BaseConfInfo.DingClientSecret, "ding_client_secret", "", "Dingding app secret")
	flag.StringVar(&BaseConfInfo.ComWechatToken, "com_wechat_token", "", "ComWechat token")
	flag.StringVar(&BaseConfInfo.ComWechatEncodingAESKey, "com_wechat_encoding_aes_key", "", "ComWechat encoding aes key")
	flag.StringVar(&BaseConfInfo.ComWechatCorpID, "com_wechat_corp_id", "", "ComWechat corp id")
	flag.StringVar(&BaseConfInfo.ComWechatSecret, "com_wechat_secret", "", "ComWechat secret")
	flag.StringVar(&BaseConfInfo.ComWechatAgentID, "com_wechat_agent_id", "", "ComWechat agent id")
	flag.StringVar(&BaseConfInfo.WechatAppID, "wechat_app_id", "", "Wechat app id")
	flag.StringVar(&BaseConfInfo.WechatAppSecret, "wechat_app_secret", "", "Wechat app secret")
	flag.StringVar(&BaseConfInfo.WechatEncodingAESKey, "wechat_encoding_aes_key", "", "Wechat encoding aes key")
	flag.StringVar(&BaseConfInfo.WechatToken, "wechat_token", "", "Wechat token")
	flag.BoolVar(&BaseConfInfo.WechatActive, "wechat_active", false, "Wechat active")
	flag.StringVar(&BaseConfInfo.QQAppID, "qq_app_id", "", "QQ app id")
	flag.StringVar(&BaseConfInfo.QQAppSecret, "qq_app_secret", "", "QQ app secret")
	flag.StringVar(&BaseConfInfo.QQOneBotReceiveToken, "qq_one_bot_receive_token", "TinyClaw", "onebot receive token")
	flag.StringVar(&BaseConfInfo.QQOneBotSendToken, "qq_one_bot_send_token", "TinyClaw", "onebot send token")
	flag.StringVar(&BaseConfInfo.QQOneBotHttpServer, "qq_one_bot_http_server", "http://127.0.0.1:3000", "onebot http server")
	flag.BoolVar(&BaseConfInfo.SmartMode, "smart_mode", false, "Smart mode")
	flag.IntVar(&BaseConfInfo.ContextExpireTime, "context_expire_time", 86400, "Context expire time")

	flag.StringVar(&BaseConfInfo.DeepseekToken, "deepseek_token", "", "deepseek auth token")
	flag.StringVar(&BaseConfInfo.OpenAIToken, "openai_token", "", "openai auth token")
	flag.StringVar(&BaseConfInfo.GeminiToken, "gemini_token", "", "gemini auth token")
	flag.StringVar(&BaseConfInfo.OpenRouterToken, "open_router_token", "", "openrouter auth token")
	flag.StringVar(&BaseConfInfo.AI302Token, "ai_302_token", "", "302.ai token")
	flag.StringVar(&BaseConfInfo.VolToken, "vol_token", "", "vol auth token")
	flag.StringVar(&BaseConfInfo.AliyunToken, "aliyun_token", "", "aliyun auth token")
	flag.StringVar(&BaseConfInfo.ErnieAK, "ernie_ak", "", "ernie ak")
	flag.StringVar(&BaseConfInfo.ErnieSK, "ernie_sk", "", "ernie sk")
	flag.StringVar(&BaseConfInfo.VolcAK, "volc_ak", "", "volc ak")
	flag.StringVar(&BaseConfInfo.VolcSK, "volc_sk", "", "volc sk")
	flag.StringVar(&BaseConfInfo.ChatAnyWhereToken, "chat_any_where_token", "", "chatAnyWhere Token")

	flag.StringVar(&BaseConfInfo.BotName, "bot_name", "TinyClaw", "bot name")
	flag.StringVar(&BaseConfInfo.CustomUrl, "custom_url", "", "custom url")
	flag.StringVar(&BaseConfInfo.CustomPath, "custom_path", "", "custom path")
	flag.StringVar(&BaseConfInfo.Type, "type", "", "llm type: deepseek gemini openai openrouter vol chatanywhere")
	flag.StringVar(&BaseConfInfo.MediaType, "media_type", "", "media type: vol gemini openai aliyun 302-ai openrouter")
	flag.StringVar(&BaseConfInfo.DBType, "db_type", "sqlite3", "db type")
	flag.StringVar(&BaseConfInfo.DBConf, "db_conf", GetAbsPath("data/tiny_claw.db"), "db conf")
	flag.StringVar(&BaseConfInfo.LLMProxy, "llm_proxy", "", "llm proxy: http://127.0.0.1:7890")
	flag.StringVar(&BaseConfInfo.RobotProxy, "robot_proxy", "", "robot proxy: http://127.0.0.1:7890")
	flag.StringVar(&BaseConfInfo.Lang, "lang", "en", "lang")
	flag.IntVar(&BaseConfInfo.TokenPerUser, "token_per_user", 100000, "token per user")
	flag.IntVar(&BaseConfInfo.MaxUserChat, "max_user_chat", 2, "max chat per user")
	flag.StringVar(&BaseConfInfo.HTTPHost, "http_host", ":36060", "http server port")
	flag.BoolVar(&BaseConfInfo.UseTools, "use_tools", false, "use function tools")
	flag.IntVar(&BaseConfInfo.MaxQAPair, "max_qa_pari", 100, "max qa pair")
	flag.StringVar(&BaseConfInfo.Character, "character", "", "ai's character")
	flag.StringVar(&BaseConfInfo.Powered, "powered", "", "powered by")
	flag.StringVar(&BaseConfInfo.ImagePath, "image_path", "./conf/img/", "image path")

	flag.StringVar(&BaseConfInfo.CrtFile, "crt_file", "", "public key file")
	flag.StringVar(&BaseConfInfo.KeyFile, "key_file", "", "secret key file")
	flag.StringVar(&BaseConfInfo.CaFile, "ca_file", "", "ca file")
	flag.BoolVar(&BaseConfInfo.SendMcpRes, "send_mcp_res", false, "send mcp res")
	flag.BoolVar(&BaseConfInfo.SendMcpMediaToLLM, "send_mcp_media_to_llm", false, "send mcp media to llm")
	flag.StringVar(&BaseConfInfo.DefaultModel, "default_model", "", "default model")
	flag.IntVar(&BaseConfInfo.LLMRetryTimes, "llm_retry_times", 3, "llm retry times")
	flag.IntVar(&BaseConfInfo.LLMRetryInterval, "llm_retry_interval", 100, "llm retry interval")
	flag.BoolVar(&BaseConfInfo.LLMOptionParam, "llm_option_param", false, "llm option param")
	flag.BoolVar(&BaseConfInfo.IsStreaming, "is_streaming", false, "is streaming")

	allowedUserIds := flag.String("allowed_user_ids", "", "allowed user ids")
	allowedGroupIds := flag.String("allowed_group_ids", "", "allowed group ids")
	privilegedUserIds := flag.String("privileged_user_ids", "", "privileged user ids")

	BaseConfInfo.AllowedUserIds = make(map[string]bool)
	BaseConfInfo.AllowedGroupIds = make(map[string]bool)
	BaseConfInfo.PrivilegedUserIds = make(map[string]bool)

	InitLLMConf()
	InitPhotoConf()
	InitVideoConf()
	InitAudioConf()
	InitToolsConf()
	InitFeatureConf()
	InitKnowledgeConf()
	InitRegisterConf()

	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)
	flag.Parse()
	allowedUserIdsValue, allowedGroupIdsValue, privilegedUserIdsValue := applyEnvOverrides(*allowedUserIds, *allowedGroupIds, *privilegedUserIds)
	normalizeLoadedConfPaths()
	logConf(allowedUserIdsValue, allowedGroupIdsValue, privilegedUserIdsValue)
	SaveConf()

}

func applyEnvOverrides(allowedUserIds, allowedGroupIds, privilegedUserIds string) (string, string, string) {
	if os.Getenv("ALLOWED_USER_IDS") != "" {
		allowedUserIds = os.Getenv("ALLOWED_USER_IDS")
	}
	if os.Getenv("ALLOWED_GROUP_IDS") != "" {
		allowedGroupIds = os.Getenv("ALLOWED_GROUP_IDS")
	}
	if os.Getenv("PRIVILEGED_USER_IDS") != "" {
		privilegedUserIds = os.Getenv("PRIVILEGED_USER_IDS")
	} else if os.Getenv("ADMIN_USER_IDS") != "" {
		privilegedUserIds = os.Getenv("ADMIN_USER_IDS")
	}

	if BaseConfInfo.AllowedUserIds == nil {
		BaseConfInfo.AllowedUserIds = make(map[string]bool)
	}
	if BaseConfInfo.AllowedGroupIds == nil {
		BaseConfInfo.AllowedGroupIds = make(map[string]bool)
	}
	if BaseConfInfo.PrivilegedUserIds == nil {
		BaseConfInfo.PrivilegedUserIds = make(map[string]bool)
	}

	if os.Getenv("TELEGRAM_BOT_TOKEN") != "" {
		BaseConfInfo.TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	if os.Getenv("CHAT_ANY_WHERE_TOKEN") != "" {
		BaseConfInfo.ChatAnyWhereToken = os.Getenv("CHAT_ANY_WHERE_TOKEN")
	}

	if os.Getenv("DISCORD_BOT_TOKEN") != "" {
		BaseConfInfo.DiscordBotToken = os.Getenv("DISCORD_BOT_TOKEN")
	}

	if os.Getenv("SLACK_BOT_TOKEN") != "" {
		BaseConfInfo.SlackBotToken = os.Getenv("SLACK_BOT_TOKEN")
	}

	if os.Getenv("SLACK_APP_TOKEN") != "" {
		BaseConfInfo.SlackAppToken = os.Getenv("SLACK_APP_TOKEN")
	}

	if os.Getenv("LARK_APP_ID") != "" {
		BaseConfInfo.LarkAPPID = os.Getenv("LARK_APP_ID")
	}

	if os.Getenv("LARK_APP_SECRET") != "" {
		BaseConfInfo.LarkAppSecret = os.Getenv("LARK_APP_SECRET")
	}

	if os.Getenv("DING_CLIENT_ID") != "" {
		BaseConfInfo.DingClientId = os.Getenv("DING_CLIENT_ID")
	}

	if os.Getenv("DING_CLIENT_SECRET") != "" {
		BaseConfInfo.DingClientSecret = os.Getenv("DING_CLIENT_SECRET")
	}

	if os.Getenv("COM_WECHAT_TOKEN") != "" {
		BaseConfInfo.ComWechatToken = os.Getenv("COM_WECHAT_TOKEN")
	}

	if os.Getenv("WECHAT_TOKEN") != "" {
		BaseConfInfo.WechatToken = os.Getenv("WECHAT_TOKEN")
	}

	if os.Getenv("WECHAT_APP_ID") != "" {
		BaseConfInfo.WechatAppID = os.Getenv("WECHAT_APP_ID")
	}

	if os.Getenv("WECHAT_APP_SECRET") != "" {
		BaseConfInfo.WechatAppSecret = os.Getenv("WECHAT_APP_SECRET")
	}

	if os.Getenv("WECHAT_ENCODING_AES_KEY") != "" {
		BaseConfInfo.WechatEncodingAESKey = os.Getenv("WECHAT_ENCODING_AES_KEY")
	}

	if os.Getenv("WECHAT_ACTIVE") != "" {
		BaseConfInfo.WechatActive = os.Getenv("WECHAT_ACTIVE") == "true"
	}

	if os.Getenv("COM_WECHAT_ENCODING_AES_KEY") != "" {
		BaseConfInfo.ComWechatEncodingAESKey = os.Getenv("COM_WECHAT_ENCODING_AES_KEY")
	}

	if os.Getenv("COM_WECHAT_CORP_ID") != "" {
		BaseConfInfo.ComWechatCorpID = os.Getenv("COM_WECHAT_CORP_ID")
	}

	if os.Getenv("COM_WECHAT_SECRET") != "" {
		BaseConfInfo.ComWechatSecret = os.Getenv("COM_WECHAT_SECRET")
	}

	if os.Getenv("COM_WECHAT_AGENT_ID") != "" {
		BaseConfInfo.ComWechatAgentID = os.Getenv("COM_WECHAT_AGENT_ID")
	}

	if os.Getenv("QQ_APP_ID") != "" {
		BaseConfInfo.QQAppID = os.Getenv("QQ_APP_ID")
	}

	if os.Getenv("QQ_APP_SECRET") != "" {
		BaseConfInfo.QQAppSecret = os.Getenv("QQ_APP_SECRET")
	}

	if os.Getenv("QQ_ONEBOT_SEND_TOKEN") != "" {
		BaseConfInfo.QQOneBotSendToken = os.Getenv("QQ_ONEBOT_SEND_TOKEN")
	}

	if os.Getenv("QQ_ONEBOT_RECEIVE_TOKEN") != "" {
		BaseConfInfo.QQOneBotReceiveToken = os.Getenv("QQ_ONEBOT_RECEIVE_TOKEN")
	}

	if os.Getenv("QQ_ONEBOT_HTTP_SERVER") != "" {
		BaseConfInfo.QQOneBotHttpServer = os.Getenv("QQ_ONEBOT_HTTP_SERVER")
	}

	if os.Getenv("DEEPSEEK_TOKEN") != "" {
		BaseConfInfo.DeepseekToken = os.Getenv("DEEPSEEK_TOKEN")
	}

	if os.Getenv("CUSTOM_URL") != "" {
		BaseConfInfo.CustomUrl = os.Getenv("CUSTOM_URL")
	}

	if os.Getenv("BOT_NAME") != "" {
		BaseConfInfo.BotName = os.Getenv("BOT_NAME")
	}

	if os.Getenv("TYPE") != "" {
		BaseConfInfo.Type = os.Getenv("TYPE")
	}

	if os.Getenv("VOLC_AK") != "" {
		BaseConfInfo.VolcAK = os.Getenv("VOLC_AK")
	}

	if os.Getenv("VOLC_SK") != "" {
		BaseConfInfo.VolcSK = os.Getenv("VOLC_SK")
	}

	if os.Getenv("DB_TYPE") != "" {
		BaseConfInfo.DBType = os.Getenv("DB_TYPE")
	}

	if os.Getenv("DB_CONF") != "" {
		BaseConfInfo.DBConf = os.Getenv("DB_CONF")
	}

	if allowedUserIds != "" {
		BaseConfInfo.AllowedUserIds = make(map[string]bool)
	}

	if allowedGroupIds != "" {
		BaseConfInfo.AllowedGroupIds = make(map[string]bool)
	}

	if privilegedUserIds != "" {
		BaseConfInfo.PrivilegedUserIds = make(map[string]bool)
	}

	if os.Getenv("LLM_PROXY") != "" {
		BaseConfInfo.LLMProxy = os.Getenv("LLM_PROXY")
	}

	if os.Getenv("ROBOT_PROXY") != "" {
		BaseConfInfo.RobotProxy = os.Getenv("ROBOT_PROXY")
	}

	if os.Getenv("LANG") != "" {
		BaseConfInfo.Lang = os.Getenv("LANG")
	}

	if os.Getenv("TOKEN_PER_USER") != "" {
		BaseConfInfo.TokenPerUser, _ = strconv.Atoi(os.Getenv("TOKEN_PER_USER"))
	}

	if os.Getenv("MAX_USER_CHAT") != "" {
		BaseConfInfo.MaxUserChat, _ = strconv.Atoi(os.Getenv("MAX_USER_CHAT"))
	}

	if os.Getenv("HTTP_HOST") != "" {
		BaseConfInfo.HTTPHost = os.Getenv("HTTP_HOST")
	}

	if os.Getenv("USE_TOOLS") != "" {
		BaseConfInfo.UseTools = os.Getenv("USE_TOOLS") == "true"
	}

	if os.Getenv("OPENAI_TOKEN") != "" {
		BaseConfInfo.OpenAIToken = os.Getenv("OPENAI_TOKEN")
	}

	if os.Getenv("GEMINI_TOKEN") != "" {
		BaseConfInfo.GeminiToken = os.Getenv("GEMINI_TOKEN")
	}

	if os.Getenv("VOL_TOKEN") != "" {
		BaseConfInfo.VolToken = os.Getenv("VOL_TOKEN")
	}

	if os.Getenv("ALIYUN_TOKEN") != "" {
		BaseConfInfo.AliyunToken = os.Getenv("ALIYUN_TOKEN")
	}

	if os.Getenv("ERNIE_AK") != "" {
		BaseConfInfo.ErnieAK = os.Getenv("ERNIE_AK")
	}

	if os.Getenv("ERNIE_SK") != "" {
		BaseConfInfo.ErnieSK = os.Getenv("ERNIE_SK")
	}

	if os.Getenv("OPEN_ROUTER_TOKEN") != "" {
		BaseConfInfo.OpenRouterToken = os.Getenv("OPEN_ROUTER_TOKEN")
	}

	if os.Getenv("AI_302_TOKEN") != "" {
		BaseConfInfo.AI302Token = os.Getenv("AI_302_TOKEN")
	}

	if os.Getenv("MAX_QA_PAIR") != "" {
		BaseConfInfo.MaxQAPair, _ = strconv.Atoi(os.Getenv("MAX_QA_PAIR"))
	}

	if os.Getenv("CHARACTER") != "" {
		BaseConfInfo.Character = os.Getenv("CHARACTER")
	}

	if os.Getenv("CRT_FILE") != "" {
		BaseConfInfo.CrtFile = os.Getenv("CRT_FILE")
	}

	if os.Getenv("KEY_FILE") != "" {
		BaseConfInfo.KeyFile = os.Getenv("KEY_FILE")
	}

	if os.Getenv("CA_FILE") != "" {
		BaseConfInfo.CaFile = os.Getenv("CA_FILE")
	}

	if os.Getenv("MEDIA_TYPE") != "" {
		BaseConfInfo.MediaType = os.Getenv("MEDIA_TYPE")
	}

	if os.Getenv("SMART_MODE") != "" {
		BaseConfInfo.SmartMode = os.Getenv("SMART_MODE") == "true"
	}

	if os.Getenv("CONTEXT_EXPIRE_TIME") != "" {
		BaseConfInfo.ContextExpireTime, _ = strconv.Atoi(os.Getenv("CONTEXT_EXPIRE_TIME"))
	}

	if os.Getenv("POWERED") != "" {
		BaseConfInfo.Powered = os.Getenv("POWERED")
	}

	if os.Getenv("SEND_MCP_RES") != "" {
		BaseConfInfo.SendMcpRes = os.Getenv("SEND_MCP_RES") == "true"
	}

	if os.Getenv("DEFAULT_MODEL") != "" {
		BaseConfInfo.DefaultModel = os.Getenv("DEFAULT_MODEL")
	}

	if os.Getenv("LLM_RETRY_TIMES") != "" {
		BaseConfInfo.LLMRetryTimes, _ = strconv.Atoi(os.Getenv("LLM_RETRY_TIMES"))
	}

	if os.Getenv("LLM_RETRY_INTERVAL") != "" {
		BaseConfInfo.LLMRetryInterval, _ = strconv.Atoi(os.Getenv("LLM_RETRY_INTERVAL"))
	}

	if os.Getenv("LLM_OPTION_PARAM") != "" {
		BaseConfInfo.LLMOptionParam = os.Getenv("LLM_OPTION_PARAM") == "true"
	}

	if os.Getenv("IMAGE_PATH") != "" {
		BaseConfInfo.ImagePath = os.Getenv("IMAGE_PATH")
	}

	if os.Getenv("IS_STREAMING") != "" {
		BaseConfInfo.IsStreaming = os.Getenv("IS_STREAMING") == "true"
	}

	if os.Getenv("SEND_MCP_MEDIA_TO_LLM") == "true" {
		BaseConfInfo.SendMcpMediaToLLM = true
	}

	EnvAudioConf()
	EnvFeatureConf()
	EnvKnowledgeConf()
	EnvLLMConf()
	EnvPhotoConf()
	EnvToolsConf()
	EnvVideoConf()
	EnvRegisterConf()

	return allowedUserIds, allowedGroupIds, privilegedUserIds
}

func logConf(allowedUserIds, allowedGroupIds, privilegedUserIds string) {
	for _, userIdStr := range strings.Split(allowedUserIds, ",") {
		if userIdStr == "" {
			continue
		}
		BaseConfInfo.AllowedUserIds[userIdStr] = true
	}

	for _, groupIdStr := range strings.Split(allowedGroupIds, ",") {
		if groupIdStr == "" {
			continue
		}
		BaseConfInfo.AllowedGroupIds[groupIdStr] = true
	}

	for _, userIdStr := range strings.Split(privilegedUserIds, ",") {
		userIdStr = strings.TrimSpace(userIdStr)
		if userIdStr == "" {
			continue
		}
		BaseConfInfo.PrivilegedUserIds[userIdStr] = true
	}

	logger.Info("CONF", "TelegramBotToken", maskSecret(BaseConfInfo.TelegramBotToken))
	logger.Info("CONF", "DiscordBotToken", maskSecret(BaseConfInfo.DiscordBotToken))
	logger.Info("CONF", "SlackBotToken", maskSecret(BaseConfInfo.SlackBotToken))
	logger.Info("CONF", "SlackAppToken", maskSecret(BaseConfInfo.SlackAppToken))
	logger.Info("CONF", "LarkAPPID", maskSecret(BaseConfInfo.LarkAPPID))
	logger.Info("CONF", "LarkAppSecret", maskSecret(BaseConfInfo.LarkAppSecret))
	logger.Info("CONF", "DingClientId", maskSecret(BaseConfInfo.DingClientId))
	logger.Info("CONF", "DingClientSecret", maskSecret(BaseConfInfo.DingClientSecret))
	logger.Info("CONF", "ComWechatToken", maskSecret(BaseConfInfo.ComWechatToken))
	logger.Info("CONF", "ComWechatEncodingAESKey", maskSecret(BaseConfInfo.ComWechatEncodingAESKey))
	logger.Info("CONF", "ComWechatCorpID", maskSecret(BaseConfInfo.ComWechatCorpID))
	logger.Info("CONF", "ComWechatSecret", maskSecret(BaseConfInfo.ComWechatSecret))
	logger.Info("CONF", "ComWechatAgentID", maskSecret(BaseConfInfo.ComWechatAgentID))
	logger.Info("CONF", "WechatToken", maskSecret(BaseConfInfo.WechatToken))
	logger.Info("CONF", "WechatAppSecret", maskSecret(BaseConfInfo.WechatAppSecret))
	logger.Info("CONF", "WechatAppID", maskSecret(BaseConfInfo.WechatAppID))
	logger.Info("CONF", "WechatActive", BaseConfInfo.WechatActive)
	logger.Info("CONF", "WechatEncodingAESKey", maskSecret(BaseConfInfo.WechatEncodingAESKey))
	logger.Info("CONF", "QQAppID", maskSecret(BaseConfInfo.QQAppID))
	logger.Info("CONF", "QQAppSecret", maskSecret(BaseConfInfo.QQAppSecret))
	logger.Info("CONF", "QQOneBotHttpServer", BaseConfInfo.QQOneBotHttpServer)
	logger.Info("CONF", "QQOneBotReceiveToken", maskSecret(BaseConfInfo.QQOneBotReceiveToken))
	logger.Info("CONF", "QQOneBotSendToken", maskSecret(BaseConfInfo.QQOneBotSendToken))
	logger.Info("CONF", "DeepseekToken", maskSecret(BaseConfInfo.DeepseekToken))
	logger.Info("CONF", "CustomUrl", BaseConfInfo.CustomUrl)
	logger.Info("CONF", "Type", BaseConfInfo.Type)
	logger.Info("CONF", "VolcAK", maskSecret(BaseConfInfo.VolcAK))
	logger.Info("CONF", "VolcSK", maskSecret(BaseConfInfo.VolcSK))
	logger.Info("CONF", "AliyunToken", maskSecret(BaseConfInfo.AliyunToken))
	logger.Info("CONF", "DBType", BaseConfInfo.DBType)
	logger.Info("CONF", "DBConf", maskMaybeDSN(BaseConfInfo.DBConf))
	logger.Info("CONF", "AllowedUserIdsCount", len(BaseConfInfo.AllowedUserIds))
	logger.Info("CONF", "AllowedGroupIdsCount", len(BaseConfInfo.AllowedGroupIds))
	logger.Info("CONF", "PrivilegedUserIdsCount", len(BaseConfInfo.PrivilegedUserIds))
	logger.Info("CONF", "LLMProxy", BaseConfInfo.LLMProxy)
	logger.Info("CONF", "RobotProxy", BaseConfInfo.RobotProxy)
	logger.Info("CONF", "Lang", BaseConfInfo.Lang)
	logger.Info("CONF", "TokenPerUser", BaseConfInfo.TokenPerUser)
	logger.Info("CONF", "MaxUserChat", BaseConfInfo.MaxUserChat)
	logger.Info("CONF", "HTTPHost", BaseConfInfo.HTTPHost)
	logger.Info("CONF", "UseTools", BaseConfInfo.UseTools)
	logger.Info("CONF", "OpenAIToken", maskSecret(BaseConfInfo.OpenAIToken))
	logger.Info("CONF", "GeminiToken", maskSecret(BaseConfInfo.GeminiToken))
	logger.Info("CONF", "OpenRouterToken", maskSecret(BaseConfInfo.OpenRouterToken))
	logger.Info("CONF", "AI302Token", maskSecret(BaseConfInfo.AI302Token))
	logger.Info("CONF", "ErnieAK", maskSecret(BaseConfInfo.ErnieAK))
	logger.Info("CONF", "ErnieSK", maskSecret(BaseConfInfo.ErnieSK))
	logger.Info("CONF", "VolToken", maskSecret(BaseConfInfo.VolToken))
	logger.Info("CONF", "CrtFile", BaseConfInfo.CrtFile)
	logger.Info("CONF", "KeyFile", BaseConfInfo.KeyFile)
	logger.Info("CONF", "CaFile", BaseConfInfo.CaFile)
	logger.Info("CONF", "MediaType", BaseConfInfo.MediaType)
	logger.Info("CONF", "BotName", BaseConfInfo.BotName)
	logger.Info("CONF", "MaxQAPair", BaseConfInfo.MaxQAPair)
	logger.Info("CONF", "SmartMode", BaseConfInfo.SmartMode)
	logger.Info("CONF", "Powered", BaseConfInfo.Powered)
	logger.Info("CONF", "Character", BaseConfInfo.Character)
	logger.Info("CONF", "ContextExpireTime", BaseConfInfo.ContextExpireTime)
	logger.Info("CONF", "SendMcpRes", BaseConfInfo.SendMcpRes)
	logger.Info("CONF", "DefaultModel", BaseConfInfo.DefaultModel)
	logger.Info("CONF", "LLMRetryTimes", BaseConfInfo.LLMRetryTimes)
	logger.Info("CONF", "LLMRetryInterval", BaseConfInfo.LLMRetryInterval)
	logger.Info("CONF", "LLMOptionParam", BaseConfInfo.LLMOptionParam)
	logger.Info("CONF", "ImagePath", BaseConfInfo.ImagePath)
	logger.Info("CONF", "IsStreaming", BaseConfInfo.IsStreaming)
	logger.Info("CONF", "SendMcpMediaToLLM", BaseConfInfo.SendMcpMediaToLLM)

	logger.Info("FEATURE_CONF", "Knowledge", FeatureConfInfo.Knowledge)
	logger.Info("FEATURE_CONF", "Media", FeatureConfInfo.Media)
	logger.Info("FEATURE_CONF", "Cron", FeatureConfInfo.Cron)
	logger.Info("FEATURE_CONF", "LegacyBots", FeatureConfInfo.LegacyBots)
	logger.Info("FEATURE_CONF", "LegacyMCPProxy", FeatureConfInfo.LegacyMCPProxy)
	logger.Info("FEATURE_CONF", "LegacyTaskTools", FeatureConfInfo.LegacyTaskTools)
	logger.Info("FEATURE_CONF", "Workflow", FeatureConfInfo.Workflow)

	logger.Info("AUDIO_CONF", "AudioAppID", maskSecret(AudioConfInfo.VolAudioAppID))
	logger.Info("AUDIO_CONF", "AudioToken", maskSecret(AudioConfInfo.VolAudioToken))
	logger.Info("AUDIO_CONF", "AudioCluster", AudioConfInfo.VolAudioRecCluster)
	logger.Info("AUDIO_CONF", "AudioVoiceType", AudioConfInfo.VolAudioVoiceType)
	logger.Info("AUDIO_CONF", "AudioTTSCluster", AudioConfInfo.VolAudioTTSCluster)
	logger.Info("AUDIO_CONF", "GeminiAudioModel", AudioConfInfo.GeminiAudioModel)
	logger.Info("AUDIO_CONF", "GeminiVoiceName", AudioConfInfo.GeminiVoiceName)
	logger.Info("AUDIO_CONF", "OpenAIAudioModel", AudioConfInfo.OpenAIAudioModel)
	logger.Info("AUDIO_CONF", "OpenAIVoiceName", AudioConfInfo.OpenAIVoiceName)
	logger.Info("AUDIO_CONF", "TTSType", AudioConfInfo.TTSType)
	logger.Info("AUDIO_CONF", "VolEndSmoothWindow", AudioConfInfo.VolEndSmoothWindow)
	logger.Info("AUDIO_CONF", "VolTTSSpeaker", AudioConfInfo.VolTTSSpeaker)
	logger.Info("AUDIO_CONF", "VolBotName", AudioConfInfo.VolBotName)
	logger.Info("AUDIO_CONF", "VolSystemRole", AudioConfInfo.VolSystemRole)
	logger.Info("AUDIO_CONF", "VolSpeakingStyle", AudioConfInfo.VolSpeakingStyle)
	logger.Info("AUDIO_CONF", "AliyunAudioModel", AudioConfInfo.AliyunAudioModel)
	logger.Info("AUDIO_CONF", "AliyunAudioVoice", AudioConfInfo.AliyunAudioVoice)
	logger.Info("AUDIO_CONF", "AliyunAudioRecModel", AudioConfInfo.AliyunAudioRecModel)

	logger.Info("KNOWLEDGE_CONF", "EmbeddingType", KnowledgeConfInfo.EmbeddingType)
	logger.Info("KNOWLEDGE_CONF", "EmbeddingBaseURL", KnowledgeConfInfo.EmbeddingBaseURL)
	logger.Info("KNOWLEDGE_CONF", "EmbeddingModelID", KnowledgeConfInfo.EmbeddingModelID)
	logger.Info("KNOWLEDGE_CONF", "EmbeddingQueryInstruction", KnowledgeConfInfo.EmbeddingQueryInstruction)
	logger.Info("KNOWLEDGE_CONF", "KnowledgePath", KnowledgeConfInfo.KnowledgePath)
	logger.Info("KNOWLEDGE_CONF", "PostgresDSN", maskSecret(KnowledgeConfInfo.PostgresDSN))
	logger.Info("KNOWLEDGE_CONF", "RedisAddr", KnowledgeConfInfo.RedisAddr)
	logger.Info("KNOWLEDGE_CONF", "MinIOEndpoint", KnowledgeConfInfo.MinIOEndpoint)
	logger.Info("KNOWLEDGE_CONF", "MinIOBucket", KnowledgeConfInfo.MinIOBucket)
	logger.Info("KNOWLEDGE_CONF", "DefaultKnowledgeBase", KnowledgeConfInfo.DefaultKnowledgeBase)
	logger.Info("KNOWLEDGE_CONF", "DefaultCollection", KnowledgeConfInfo.DefaultCollection)

	logger.Info("PHOTO_CONF", "ReqKey", PhotoConfInfo.ReqKey)
	logger.Info("PHOTO_CONF", "ModelVersion", PhotoConfInfo.ModelVersion)
	logger.Info("PHOTO_CONF", "ReqScheduleConf", PhotoConfInfo.ReqScheduleConf)
	logger.Info("PHOTO_CONF", "Seed", PhotoConfInfo.Seed)
	logger.Info("PHOTO_CONF", "Width", PhotoConfInfo.Width)
	logger.Info("PHOTO_CONF", "Height", PhotoConfInfo.Height)
	logger.Info("PHOTO_CONF", "Scale", PhotoConfInfo.Scale)
	logger.Info("PHOTO_CONF", "DDIMSteps", PhotoConfInfo.DDIMSteps)
	logger.Info("PHOTO_CONF", "UsePreLLM", PhotoConfInfo.UsePreLLM)
	logger.Info("PHOTO_CONF", "UseSr", PhotoConfInfo.UseSr)
	logger.Info("PHOTO_CONF", "ReturnUrl", PhotoConfInfo.ReturnUrl)
	logger.Info("PHOTO_CONF", "AddLogo", PhotoConfInfo.AddLogo)
	logger.Info("PHOTO_CONF", "Position", PhotoConfInfo.Position)
	logger.Info("PHOTO_CONF", "Language", PhotoConfInfo.Language)
	logger.Info("PHOTO_CONF", "Opacity", PhotoConfInfo.Opacity)
	logger.Info("PHOTO_CONF", "LogoTextContent", PhotoConfInfo.LogoTextContent)
	logger.Info("PHOTO_CONF", "GeminiImageModel", PhotoConfInfo.GeminiImageModel)
	logger.Info("PHOTO_CONF", "GeminiRecModel", PhotoConfInfo.GeminiRecModel)
	logger.Info("PHOTO_CONF", "OpenAIImageStyle", PhotoConfInfo.OpenAIImageStyle)
	logger.Info("PHOTO_CONF", "OpenAIImageModel", PhotoConfInfo.OpenAIImageModel)
	logger.Info("PHOTO_CONF", "OpenAIImageSize", PhotoConfInfo.OpenAIImageSize)
	logger.Info("PHOTO_CONF", "OpenAIRecModel", PhotoConfInfo.OpenAIRecModel)
	logger.Info("PHOTO_CONF", "VolImageModel", PhotoConfInfo.VolImageModel)
	logger.Info("PHOTO_CONF", "VolRecModel", PhotoConfInfo.VolRecModel)
	logger.Info("PHOTO_CONF", "AI302ImageModel", PhotoConfInfo.MixRecModel)
	logger.Info("PHOTO_CONF", "AI302RecModel", PhotoConfInfo.MixRecModel)
	logger.Info("PHOTO_CONF", "AliyunImageModel", PhotoConfInfo.AliyunImageModel)
	logger.Info("PHOTO_CONF", "AliyunRecModel", PhotoConfInfo.AliyunRecModel)

	logger.Info("VIDEO_CONF", "VOL_VIDEO_MODEL", VideoConfInfo.VolVideoModel)
	logger.Info("VIDEO_CONF", "RADIO", VideoConfInfo.Radio)
	logger.Info("VIDEO_CONF", "DURATION", VideoConfInfo.Duration)
	logger.Info("VIDEO_CONF", "FPS", VideoConfInfo.FPS)
	logger.Info("VIDEO_CONF", "RESOLUTION", VideoConfInfo.Resolution)
	logger.Info("VIDEO_CONF", "WATERMARK", VideoConfInfo.Watermark)
	logger.Info("AUDIO_CONF", "GeminiVideoModel", VideoConfInfo.GeminiVideoModel)
	logger.Info("AUDIO_CONF", "AI302VideoModel", VideoConfInfo.AI302VideoModel)
	logger.Info("AUDIO_CONF", "AliyunVideoModel", VideoConfInfo.AliyunVideoModel)

	logger.Info("REGISTER_CONF", "Type", RegisterConfInfo.Type)
	logger.Info("REGISTER_CONF", "EtcdURLs", RegisterConfInfo.EtcdURLs)
	logger.Info("REGISTER_CONF", "EtcdUsername", maskSecret(RegisterConfInfo.EtcdUsername))
	logger.Info("REGISTER_CONF", "EtcdPassword", maskSecret(RegisterConfInfo.EtcdPassword))

	logger.Info("LLM_CONF", "FrequencyPenalty", LLMConfInfo.FrequencyPenalty)
	logger.Info("LLM_CONF", "MaxTokens", LLMConfInfo.MaxTokens)
	logger.Info("LLM_CONF", "PresencePenalty", LLMConfInfo.PresencePenalty)
	logger.Info("LLM_CONF", "Temperature", LLMConfInfo.Temperature)
	logger.Info("LLM_CONF", "TopP", LLMConfInfo.TopP)
	logger.Info("LLM_CONF", "Stop", LLMConfInfo.Stop)
	logger.Info("LLM_CONF", "LogProbs", LLMConfInfo.LogProbs)
	logger.Info("LLM_CONF", "TopLogProbs", LLMConfInfo.TopLogProbs)

	logger.Info("TOOLS_CONF", "McpConfPath", *ToolsConfInfo.McpConfPath)
}

func IsPrivilegedUser(userID string) bool {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false
	}
	if BaseConfInfo == nil || len(BaseConfInfo.PrivilegedUserIds) == 0 {
		return false
	}
	return BaseConfInfo.PrivilegedUserIds[userID]
}

func GetAbsPath(relPath string) string {
	baseDir := resolveBaseDir()
	if baseDir == "" {
		return ""
	}
	return filepath.Join(baseDir, relPath)
}

func loadConf() bool {
	m := make(map[string]string)
	for _, part := range os.Args {
		if strings.HasPrefix(part, "-") {
			kv := strings.SplitN(part[1:], "=", 2)
			if len(kv) == 2 {
				m[kv[0]] = kv[1]
			}
		}
	}

	if !(len(m) == 0 || (len(m) == 1 && (m["bot_name"] != "" || m["http_host"] != "")) ||
		(len(m) == 2 && m["bot_name"] != "" && m["http_host"] != "")) {
		return false
	}

	data, err := os.ReadFile(getSaveConf(m))
	if err != nil {
		return false
	}

	err = json.Unmarshal(data, &AllConf)
	if err != nil {
		logger.Error("Failed to parse config file", "err", err)
		return false
	}

	if err = transferOptionalSection(AllConf, "base", BaseConfInfo, true); err != nil {
		logger.Error("Failed to transfer map to base conf", "err", err)
		return false
	}

	if err = transferOptionalSection(AllConf, "audio", AudioConfInfo, true); err != nil {
		logger.Error("Failed to transfer map to audio conf", "err", err)
		return false
	}

	if err = transferOptionalSection(AllConf, "llm", LLMConfInfo, true); err != nil {
		logger.Error("Failed to transfer map to llm conf", "err", err)
		return false
	}

	if err = transferOptionalSection(AllConf, "photo", PhotoConfInfo, true); err != nil {
		logger.Error("Failed to transfer map to photo conf", "err", err)
		return false
	}

	if err = transferOptionalSection(AllConf, "features", FeatureConfInfo, false); err != nil {
		logger.Error("Failed to transfer map to feature conf", "err", err)
		return false
	}

	if err = transferOptionalSection(AllConf, "knowledge", KnowledgeConfInfo, true); err != nil {
		logger.Error("Failed to transfer map to knowledge conf", "err", err)
		return false
	}

	if err = transferOptionalSection(AllConf, "video", VideoConfInfo, true); err != nil {
		logger.Error("Failed to transfer map to video conf", "err", err)
		return false
	}

	if err = transferOptionalSection(AllConf, "register", RegisterConfInfo, true); err != nil {
		logger.Error("Failed to transfer map to register conf", "err", err)
		return false
	}

	if err = transferOptionalSection(AllConf, "tools", ToolsConfInfo, true); err != nil {
		logger.Error("Failed to transfer map to tools conf", "err", err)
		return false
	}

	return true
}

func SaveConf() {
	AllConf["base"] = BaseConfInfo
	AllConf["audio"] = AudioConfInfo
	AllConf["llm"] = LLMConfInfo
	AllConf["photo"] = PhotoConfInfo
	AllConf["features"] = FeatureConfInfo
	AllConf["knowledge"] = KnowledgeConfInfo
	AllConf["video"] = VideoConfInfo
	AllConf["register"] = RegisterConfInfo
	AllConf["tools"] = ToolsConfInfo

	fileName := getSaveConf(map[string]string{
		"bot_name":  BaseConfInfo.BotName,
		"http_host": BaseConfInfo.HTTPHost,
	})

	confData, err := json.Marshal(AllConf)
	if err != nil {
		logger.Error("Failed to marshal config data", "err", err)
		return
	}

	if err = os.MkdirAll(filepath.Dir(fileName), 0700); err != nil {
		logger.Error("Failed to create config directory", "err", err)
		return
	}

	tmpFile := fileName + ".tmp"
	err = os.WriteFile(tmpFile, confData, 0600)
	if err != nil {
		logger.Error("Failed to write config file", "err", err)
		return
	}
	if err = os.Rename(tmpFile, fileName); err != nil {
		logger.Error("Failed to replace config file", "err", err)
	}

}

func getSaveConf(m map[string]string) string {
	botName := m["bot_name"]
	if botName == "" {
		botName = "TinyClaw"
	}

	httpHost := m["http_host"]
	if httpHost == "" {
		httpHost = ":36060"
	}
	httpHost = NormalizeHTTP(httpHost)

	hash := md5.Sum([]byte(httpHost))
	md5Str := hex.EncodeToString(hash[:])
	return GetAbsPath(filepath.Join("data", botName+md5Str+".json"))
}

func NormalizeHTTP(addr string) string {
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	if !strings.HasPrefix(addr, "http://") {
		addr = "http://" + addr
	}
	return addr
}

func TransferMapToConf(m map[string]interface{}, conf interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, conf)
}

func transferOptionalSection(all map[string]interface{}, key string, target interface{}, required bool) error {
	raw, ok := all[key]
	if !ok || raw == nil {
		if required {
			return os.ErrNotExist
		}
		return nil
	}

	section, ok := raw.(map[string]interface{})
	if !ok {
		return os.ErrInvalid
	}

	return TransferMapToConf(section, target)
}

func normalizeLoadedConfPaths() {
	if BaseConfInfo != nil && strings.EqualFold(strings.TrimSpace(BaseConfInfo.DBType), "sqlite3") {
		BaseConfInfo.DBConf = normalizeProjectManagedPath(BaseConfInfo.DBConf, "data", "conf")
		BaseConfInfo.ImagePath = normalizeProjectManagedPath(BaseConfInfo.ImagePath, "conf", "data")
	}

	if KnowledgeConfInfo != nil {
		KnowledgeConfInfo.KnowledgePath = normalizeProjectManagedPath(KnowledgeConfInfo.KnowledgePath, "data")
	}

	if ToolsConfInfo != nil && ToolsConfInfo.McpConfPath != nil {
		*ToolsConfInfo.McpConfPath = normalizeProjectManagedPath(*ToolsConfInfo.McpConfPath, "conf", "data")
	}
}

func normalizeProjectManagedPath(value string, roots ...string) string {
	original := strings.TrimSpace(value)
	value = original
	if value == "" || !filepath.IsAbs(value) || pathExists(value) {
		return value
	}
	if !looksLikePersistedProjectPath(value) {
		return value
	}

	baseDir := resolveBaseDir()
	if baseDir == "" {
		return value
	}

	relativePath, ok := projectRelativePath(value, roots...)
	if !ok {
		return value
	}

	relocated := filepath.Join(baseDir, relativePath)
	logger.Warn("Relocating project-managed path", "from", original, "to", relocated)
	return relocated
}

func projectRelativePath(value string, roots ...string) (string, bool) {
	if len(roots) == 0 {
		return "", false
	}

	normalized := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(value)), filepath.ToSlash(filepath.VolumeName(value)))
	normalized = strings.TrimPrefix(normalized, "/")
	parts := strings.Split(normalized, "/")
	for idx := len(parts) - 1; idx >= 0; idx-- {
		for _, root := range roots {
			if parts[idx] == root {
				return filepath.Join(parts[idx:]...), true
			}
		}
	}
	return "", false
}

func resolveBaseDir() string {
	if root := strings.TrimSpace(os.Getenv("TINYCLAW_ROOT")); root != "" {
		if abs, err := filepath.Abs(root); err == nil {
			return abs
		}
		return root
	}

	for _, candidate := range candidateBaseDirs() {
		if root := findProjectRoot(candidate); root != "" {
			return root
		}
	}

	return ""
}

func candidateBaseDirs() []string {
	candidates := make([]string, 0, 2)
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}
	return candidates
}

func findProjectRoot(start string) string {
	current := filepath.Clean(start)
	for current != "." && current != string(filepath.Separator) {
		if looksLikeProjectRoot(current) {
			return current
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}
	if looksLikeProjectRoot(current) {
		return current
	}
	return ""
}

func looksLikeProjectRoot(dir string) bool {
	if dir == "" {
		return false
	}

	if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
		if bytes.Contains(data, []byte("module github.com/LittleSongxx/TinyClaw")) {
			return true
		}
	}

	return isDir(filepath.Join(dir, "conf")) && (isDir(filepath.Join(dir, "cmd")) || isDir(filepath.Join(dir, "data")))
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func looksLikePersistedProjectPath(path string) bool {
	normalized := "/" + strings.Trim(strings.ToLower(filepath.ToSlash(filepath.Clean(path))), "/") + "/"
	return strings.Contains(normalized, "/tinyclaw/")
}
