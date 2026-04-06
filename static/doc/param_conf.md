# TinyClaw Runtime Parameters

This document keeps only the runtime parameters that are still high-signal and actively useful in the current TinyClaw project.

TinyClaw supports both:

- environment variables
- command-line flags

For the current repository layout, environment variables are the recommended choice, especially with Docker Compose.

## Recommended Workflow

Edit:

`deploy/docker/.env`

Then start the stack:

```bash
./scripts/start.sh
```

## Naming Rules

In general:

- environment variables use `UPPER_SNAKE_CASE`
- command-line flags use `lower_snake_case`

Example:

- env var: `LARK_APP_ID`
- flag: `-lark_app_id`

## High-Value Parameter Groups

### 1. Platform Access

| Variable | Purpose |
|---|---|
| `BOT_NAME` | bot display/runtime name |
| `LARK_APP_ID` | Feishu / Lark App ID |
| `LARK_APP_SECRET` | Feishu / Lark App Secret |
| `QQ_APP_ID` | QQ Open Platform App ID |
| `QQ_APP_SECRET` | QQ Open Platform App Secret |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token |

Notes:

- Feishu is the currently recommended platform
- leave unused platform credentials empty

### 2. Model and Media Providers

| Variable | Purpose |
|---|---|
| `TYPE` | text model provider |
| `DEFAULT_MODEL` | default text model |
| `MEDIA_TYPE` | media generation provider |
| `OPENAI_TOKEN` | OpenAI token |
| `GEMINI_TOKEN` | Gemini token |
| `ALIYUN_TOKEN` | Aliyun Bailian token |
| `VOL_TOKEN` | Volcano Engine token |
| `AI_302_TOKEN` | 302.AI token |

Current recommended values:

```env
TYPE=aliyun
DEFAULT_MODEL=qwen-max
MEDIA_TYPE=aliyun
```

### 3. Storage and Runtime

| Variable | Purpose |
|---|---|
| `DB_TYPE` | `sqlite3` or `mysql` |
| `DB_CONF` | database file path or DSN |
| `LANG` | runtime language, commonly `zh` or `en` |
| `HTTP_HOST` | main service listen address |
| `TOKEN_PER_USER` | per-user quota limit |
| `MAX_USER_CHAT` | max concurrent chats per user |
| `MAX_QA_PAIR` | retained QA pairs in context |
| `CHARACTER` | system persona / behavior prompt |

Recommended defaults in this repo:

```env
DB_TYPE=sqlite3
LANG=zh
HTTP_HOST=:36060
```

### 4. Admin Panel

| Variable | Purpose |
|---|---|
| `SESSION_KEY` | admin session signing key |
| `ADMIN_PORT` | admin listen port |

### 5. Proxies and HTTPS

| Variable | Purpose |
|---|---|
| `LLM_PROXY` | proxy for model requests |
| `ROBOT_PROXY` | proxy for platform requests |
| `CRT_FILE` | HTTPS certificate file |
| `KEY_FILE` | HTTPS private key |
| `CA_FILE` | CA certificate |

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

### MySQL

```env
DB_TYPE=mysql
DB_CONF=root:password@tcp(127.0.0.1:3306)/tinyclaw?charset=utf8mb4&parseTime=True&loc=Local
```

### Enable MCP / tools

```env
USE_TOOLS=true
```

## Related Docs

For deeper feature-specific settings, continue with:

- Feishu / Lark: `static/doc/lark.md`
- RAG: `static/doc/rag.md`
- Audio: `static/doc/audioconf.md`
- Photo: `static/doc/photoconf.md`
- Video: `static/doc/videoconf.md`
- Web API: `static/doc/web_api.md`
