# TinyClaw 参数说明

这份文档只保留 TinyClaw 当前仍然高频、稳定、值得维护的运行参数说明。

TinyClaw 同时支持：

- 环境变量
- 命令行参数

通常推荐优先使用环境变量，尤其是在当前仓库的 Docker Compose 结构里。

## 当前推荐方式

编辑：

`deploy/docker/.env`

然后启动：

```bash
./scripts/start.sh
```

## 参数来源规则

一般来说：

- 环境变量名：`UPPER_SNAKE_CASE`
- 命令行参数名：`lower_snake_case`

例如：

- 环境变量：`LARK_APP_ID`
- 命令行参数：`-lark_app_id`

## 最常用参数分组

### 1. 平台接入

| 变量 | 说明 |
|---|---|
| `BOT_NAME` | 机器人名称 |
| `LARK_APP_ID` | 飞书 App ID |
| `LARK_APP_SECRET` | 飞书 App Secret |
| `QQ_APP_ID` | QQ 开放平台 App ID |
| `QQ_APP_SECRET` | QQ 开放平台 App Secret |
| `TELEGRAM_BOT_TOKEN` | Telegram Bot Token |

说明：

- 当前推荐平台是飞书
- 不使用的平台凭据留空即可

### 2. 模型与媒体能力

| 变量 | 说明 |
|---|---|
| `TYPE` | 文本模型提供方 |
| `DEFAULT_MODEL` | 默认文本模型 |
| `MEDIA_TYPE` | 图片/视频能力提供方 |
| `OPENAI_TOKEN` | OpenAI Token |
| `GEMINI_TOKEN` | Gemini Token |
| `ALIYUN_TOKEN` | 阿里云百炼 Token |
| `VOL_TOKEN` | 火山引擎通用 Token |
| `AI_302_TOKEN` | 302.AI Token |

当前推荐值：

```env
TYPE=aliyun
DEFAULT_MODEL=qwen-max
MEDIA_TYPE=aliyun
```

### 3. 存储与运行

| 变量 | 说明 |
|---|---|
| `DB_TYPE` | 数据库类型，`sqlite3` 或 `mysql` |
| `DB_CONF` | 数据库文件路径或连接串 |
| `LANG` | 语言，常见为 `zh` / `en` |
| `HTTP_HOST` | 主服务监听地址 |
| `TOKEN_PER_USER` | 单用户额度限制 |
| `MAX_USER_CHAT` | 单用户最大并发会话数 |
| `MAX_QA_PAIR` | 上下文保留问答对数量 |
| `CHARACTER` | 系统人格设定 |

当前仓库默认推荐：

```env
DB_TYPE=sqlite3
LANG=zh
HTTP_HOST=:36060
```

### 4. Admin 后台

| 变量 | 说明 |
|---|---|
| `SESSION_KEY` | 后台登录态密钥 |
| `ADMIN_PORT` | 后台监听端口 |

### 5. 代理与网络

| 变量 | 说明 |
|---|---|
| `LLM_PROXY` | 模型请求代理 |
| `ROBOT_PROXY` | 平台请求代理 |
| `CRT_FILE` | HTTPS 证书 |
| `KEY_FILE` | HTTPS 私钥 |
| `CA_FILE` | CA 证书 |

## 常见配置示例

### 飞书 + Qwen

```env
BOT_NAME=TinyClawLark
LANG=zh
TYPE=aliyun
MEDIA_TYPE=aliyun
DEFAULT_MODEL=qwen-max
DB_TYPE=sqlite3

LARK_APP_ID=your_lark_app_id
LARK_APP_SECRET=your_lark_app_secret
ALIYUN_TOKEN=your_qwen_api_key
```

### 使用 MySQL

```env
DB_TYPE=mysql
DB_CONF=root:password@tcp(127.0.0.1:3306)/tinyclaw?charset=utf8mb4&parseTime=True&loc=Local
```

### 启用 MCP / Tools

```env
USE_TOOLS=true
```

## 与专题文档的关系

如果你需要深入看某一块配置，继续阅读：

- 飞书接入：`static/doc/lark_ZH.md`
- RAG：`static/doc/rag_ZH.md`
- 音频：`static/doc/audioconf_ZH.md`
- 图片：`static/doc/photoconf_ZH.md`
- 视频：`static/doc/videoconf_ZH.md`
- Web API：`static/doc/web_api_ZH.md`
