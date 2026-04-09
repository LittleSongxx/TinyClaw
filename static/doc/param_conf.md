# TinyClaw Runtime Parameters

This document keeps only the runtime parameters that are high-signal for the current TinyClaw project layout.

For the maintained Docker workflow, the main source of truth is:

```text
deploy/docker/.env
```

Most settings can also be provided as flags, but environment variables are the recommended path.

## Current Workflow

1. Copy `deploy/docker/.env.example` to `deploy/docker/.env`
2. Fill the required platform and model secrets
3. Optionally fill MCP-related secrets and knowledge settings
4. Start with `./scripts/start.sh`

## Naming Rules

- environment variables use `UPPER_SNAKE_CASE`
- flags use `lower_snake_case`

Example:

- env: `LARK_APP_ID`
- flag: `-lark_app_id`

## High-Value Parameter Groups

### 1. Platform Access

| Variable | Purpose |
|---|---|
| `BOT_NAME` | bot display/runtime name |
| `LANG` | runtime language, usually `zh` or `en` |
| `LARK_APP_ID` | Feishu / Lark App ID |
| `LARK_APP_SECRET` | Feishu / Lark App Secret |
| `QQ_APP_ID` | QQ Open Platform App ID |
| `QQ_APP_SECRET` | QQ Open Platform App Secret |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token |

Current maintained setup in this repository: Feishu / Lark.

### 2. Model and Media Providers

| Variable | Purpose |
|---|---|
| `TYPE` | text model provider |
| `DEFAULT_MODEL` | default text model |
| `MEDIA_TYPE` | image/video provider |
| `ALIYUN_TOKEN` | Aliyun Bailian token |
| `OPENAI_TOKEN` | OpenAI token |
| `GEMINI_TOKEN` | Gemini token |
| `VOL_TOKEN` | Volcano Engine token |
| `AI_302_TOKEN` | 302.AI token |

Current recommended values:

```env
TYPE=aliyun
DEFAULT_MODEL=qwen-max
MEDIA_TYPE=aliyun
```

### 3. Runtime and Admin

| Variable | Purpose |
|---|---|
| `DB_TYPE` | main app database type, usually `sqlite3` |
| `DB_CONF` | database file path or DSN |
| `HTTP_HOST` | bot HTTP listen address |
| `ADMIN_PORT` | admin listen port |
| `SESSION_KEY` | admin session signing key |
| `CHECK_BOT_SEC` | bot heartbeat / check interval |
| `LOG_LEVEL` | runtime log level |
| `TOKEN_PER_USER` | per-user token quota |
| `MAX_USER_CHAT` | max concurrent chats per user |
| `MAX_QA_PAIR` | retained QA pairs in context |
| `CHARACTER` | system persona / behavior prompt |

Current repo defaults are aligned with:

```env
DB_TYPE=sqlite3
HTTP_HOST=:36060
ADMIN_PORT=18080
LOG_LEVEL=info
```

### 4. MCP and Skills

| Variable | Purpose |
|---|---|
| `USE_TOOLS` | recommended model-side tool switch for tool-enabled deployments |
| `MCP_CONF_PATH` | optional override for the default MCP config path |
| `AMAP_MAPS_API_KEY` | secret for the AMap MCP server |
| `BOCHA_API_KEY` | secret for the Bocha search MCP server |
| `GITHUB_PERSONAL_ACCESS_TOKEN` | secret for the GitHub MCP server |

Notes:

- if `MCP_CONF_PATH` is empty, TinyClaw uses `conf/mcp/mcp.json`
- the local skill catalog is loaded from `skills/`
- current maintained Docker deployments keep `USE_TOOLS=true`

### 5. Knowledge and Embeddings

| Variable | Purpose |
|---|---|
| `EMBEDDING_TYPE` | embedding provider type |
| `EMBEDDING_BASE_URL` | embedding service URL |
| `EMBEDDING_MODEL_ID` | embedding model id |
| `EMBEDDING_QUERY_INSTRUCTION` | query-side embedding instruction |
| `EMBEDDING_DIMENSIONS` | embedding vector size |
| `CHUNK_SIZE` | document chunk size |
| `CHUNK_OVERLAP` | chunk overlap size |
| `DEFAULT_KNOWLEDGE_BASE` | default knowledge base name |
| `DEFAULT_COLLECTION` | default collection name |
| `KNOWLEDGE_AUTO_MIGRATE` | auto-migrate legacy files into the unified knowledge store |
| `RERANKER_BASE_URL` | optional reranker endpoint |

The current Docker setup uses the unified knowledge stack:

```env
EMBEDDING_TYPE=huggingface
EMBEDDING_BASE_URL=http://hf-embeddings:80
EMBEDDING_MODEL_ID=BAAI/bge-small-zh-v1.5
```

### 6. Backing Services

| Variable | Purpose |
|---|---|
| `POSTGRES_DB` | PostgreSQL database name |
| `POSTGRES_USER` | PostgreSQL username |
| `POSTGRES_PASSWORD` | PostgreSQL password |
| `POSTGRES_DSN` | PostgreSQL DSN |
| `REDIS_ADDR` | Redis address |
| `REDIS_PASSWORD` | Redis password |
| `REDIS_DB` | Redis logical database used only when async ingestion is enabled |
| `MINIO_ENDPOINT` | MinIO endpoint |
| `MINIO_ACCESS_KEY` | MinIO access key |
| `MINIO_SECRET_KEY` | MinIO secret key |
| `MINIO_BUCKET` | MinIO bucket for knowledge files |
| `MINIO_USE_SSL` | whether MinIO uses TLS |

Notes:

- `POSTGRES_DSN + MINIO_ENDPOINT` form the default unified knowledge path.
- `REDIS_*` is optional; without it, document ingestion runs synchronously.
- `etcd + milvus` are now optional containers behind `ENABLE_FULL_STACK=true` and are no longer part of the default runtime path.

## Common Examples

### Feishu + Qwen

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

### Enable MCP / Skills

```env
USE_TOOLS=true
```

### Fill the bundled MCP secrets

```env
AMAP_MAPS_API_KEY=your-amap-key
BOCHA_API_KEY=your-bocha-key
GITHUB_PERSONAL_ACCESS_TOKEN=your-github-pat
```

## Related Docs

- Admin: `static/doc/admin.md`
- MCP / Skills: `static/doc/functioncall.md`
- Knowledge: `static/doc/knowledge.md`
- Web API: `static/doc/web_api.md`
