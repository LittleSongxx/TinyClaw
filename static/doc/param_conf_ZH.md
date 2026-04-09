# TinyClaw 参数说明

这份文档只保留当前 TinyClaw 项目里最常用、最值得维护的运行参数。

对当前仓库默认的 Docker 工作流来说，主要配置入口就是：

```text
deploy/docker/.env
```

大部分参数也可以通过命令行 flag 传入，但当前项目更推荐直接维护环境变量。

## 当前推荐方式

1. 复制 `deploy/docker/.env.example` 为 `deploy/docker/.env`
2. 填写平台和模型凭据
3. 按需补齐 MCP 密钥与 Knowledge 参数
4. 执行 `./scripts/start.sh`

## 参数命名规则

- 环境变量名使用 `UPPER_SNAKE_CASE`
- 命令行参数使用 `lower_snake_case`

例如：

- 环境变量：`LARK_APP_ID`
- 命令行参数：`-lark_app_id`

## 最常用参数分组

### 1. 平台接入

| 变量 | 说明 |
|---|---|
| `BOT_NAME` | 机器人名称 |
| `LANG` | 运行语言，常见为 `zh` / `en` |
| `LARK_APP_ID` | 飞书 App ID |
| `LARK_APP_SECRET` | 飞书 App Secret |
| `QQ_APP_ID` | QQ 开放平台 App ID |
| `QQ_APP_SECRET` | QQ 开放平台 App Secret |
| `TELEGRAM_BOT_TOKEN` | Telegram Bot Token |

当前仓库主维护的平台是飞书。

### 2. 模型与媒体能力

| 变量 | 说明 |
|---|---|
| `TYPE` | 文本模型提供方 |
| `DEFAULT_MODEL` | 默认文本模型 |
| `MEDIA_TYPE` | 图片 / 视频能力提供方 |
| `ALIYUN_TOKEN` | 阿里云百炼 Token |
| `OPENAI_TOKEN` | OpenAI Token |
| `GEMINI_TOKEN` | Gemini Token |
| `VOL_TOKEN` | 火山引擎 Token |
| `AI_302_TOKEN` | 302.AI Token |

当前推荐值：

```env
TYPE=aliyun
DEFAULT_MODEL=qwen-max
MEDIA_TYPE=aliyun
```

### 3. 运行与后台

| 变量 | 说明 |
|---|---|
| `DB_TYPE` | 主库类型，通常是 `sqlite3` |
| `DB_CONF` | 数据库文件路径或 DSN |
| `HTTP_HOST` | Bot HTTP 监听地址 |
| `ADMIN_PORT` | Admin 监听端口 |
| `SESSION_KEY` | Admin 登录态签名密钥 |
| `CHECK_BOT_SEC` | Bot 检查间隔 |
| `LOG_LEVEL` | 日志级别 |
| `TOKEN_PER_USER` | 单用户额度限制 |
| `MAX_USER_CHAT` | 单用户最大并发会话数 |
| `MAX_QA_PAIR` | 上下文保留问答对数量 |
| `CHARACTER` | 系统人格设定 |

当前仓库默认配置主要围绕这组值：

```env
DB_TYPE=sqlite3
HTTP_HOST=:36060
ADMIN_PORT=18080
LOG_LEVEL=info
```

### 4. MCP 与 Skills

| 变量 | 说明 |
|---|---|
| `USE_TOOLS` | 面向工具型部署时推荐保持开启的模型侧工具开关 |
| `MCP_CONF_PATH` | 可选，自定义 MCP 配置文件路径 |
| `AMAP_MAPS_API_KEY` | AMap MCP 服务密钥 |
| `BOCHA_API_KEY` | Bocha 搜索 MCP 服务密钥 |
| `GITHUB_PERSONAL_ACCESS_TOKEN` | GitHub MCP 服务密钥 |

说明：

- 如果 `MCP_CONF_PATH` 不设置，默认读取 `conf/mcp/mcp.json`
- 本地技能目录默认从 `skills/` 加载
- 当前维护中的 Docker 方案默认保持 `USE_TOOLS=true`

### 5. Knowledge 与向量检索

| 变量 | 说明 |
|---|---|
| `EMBEDDING_TYPE` | embedding 提供方 |
| `EMBEDDING_BASE_URL` | embedding 服务地址 |
| `EMBEDDING_MODEL_ID` | embedding 模型 ID |
| `EMBEDDING_QUERY_INSTRUCTION` | query 侧 embedding 指令 |
| `EMBEDDING_DIMENSIONS` | 向量维度 |
| `CHUNK_SIZE` | 文档切块大小 |
| `CHUNK_OVERLAP` | 切块重叠大小 |
| `DEFAULT_KNOWLEDGE_BASE` | 默认知识库名 |
| `DEFAULT_COLLECTION` | 默认 collection 名 |
| `KNOWLEDGE_AUTO_MIGRATE` | 是否自动把旧文件迁移到统一知识库 |
| `RERANKER_BASE_URL` | 可选 reranker 地址 |

当前 Docker 默认值围绕统一 knowledge 栈：

```env
EMBEDDING_TYPE=huggingface
EMBEDDING_BASE_URL=http://hf-embeddings:80
EMBEDDING_MODEL_ID=BAAI/bge-small-zh-v1.5
```

### 6. 底层依赖服务

| 变量 | 说明 |
|---|---|
| `POSTGRES_DB` | PostgreSQL 数据库名 |
| `POSTGRES_USER` | PostgreSQL 用户名 |
| `POSTGRES_PASSWORD` | PostgreSQL 密码 |
| `POSTGRES_DSN` | PostgreSQL 连接串 |
| `REDIS_ADDR` | Redis 地址 |
| `REDIS_PASSWORD` | Redis 密码 |
| `REDIS_DB` | Redis 逻辑库，开启异步 ingestion 时使用 |
| `MINIO_ENDPOINT` | MinIO 地址 |
| `MINIO_ACCESS_KEY` | MinIO Access Key |
| `MINIO_SECRET_KEY` | MinIO Secret Key |
| `MINIO_BUCKET` | 知识库文件存储桶 |
| `MINIO_USE_SSL` | MinIO 是否启用 TLS |

说明：

- `POSTGRES_DSN + MINIO_ENDPOINT` 构成统一 knowledge 主路径。
- `REDIS_*` 是可选的；如果未配置，文档入库会走同步处理。
- `etcd + milvus` 只在 `ENABLE_FULL_STACK=true` 时作为可选扩展容器启动，不再是默认主链路。

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

### 启用 MCP / Skills

```env
USE_TOOLS=true
```

### 填写仓库自带 MCP 所需密钥

```env
AMAP_MAPS_API_KEY=your-amap-key
BOCHA_API_KEY=your-bocha-key
GITHUB_PERSONAL_ACCESS_TOKEN=your-github-pat
```

## 相关文档

- Admin：`static/doc/admin_ZH.md`
- MCP / Skills：`static/doc/functioncall_ZH.md`
- Knowledge：`static/doc/knowledge_ZH.md`
- Web API：`static/doc/web_api_ZH.md`
