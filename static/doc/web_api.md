# TinyClaw Web API

This document describes the current bot-side HTTP endpoints exposed by `http/http.go`.

The base URL is the bot HTTP address, for example:

```text
http://127.0.0.1:36060
```

## Response Envelope

Successful responses use:

```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

Failed responses use a non-zero `code` and an error message in `msg`.

## Core Runtime Endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/pong` | health check |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/dashboard` | runtime counters and start time |
| `GET` | `/command/get` | current effective CLI-style overrides |
| `GET` | `/conf/get` | current runtime config snapshot |
| `POST` | `/conf/update` | update one config field |
| `POST` | `/restart` | restart process with provided params |
| `POST` | `/stop` | stop current process |
| `GET` | `/log` | stream `log/tiny_claw.log` |

## Real-Time Chat Endpoint

### `POST /communicate`

This is the main SSE endpoint for text chat, image/video flows, `/task`, and `/mcp`.

Query parameters:

| Parameter | Required | Notes |
|---|---|---|
| `prompt` | yes | plain text or slash command |
| `user_id` | yes | runtime user identifier |

Request body:

- optional binary payload for image, audio, or other command-specific inputs

Response:

- `text/event-stream`

Common commands:

- `/help`
- `/clear`
- `/retry`
- `/mode`
- `/state`
- `/photo`
- `/video`
- `/task`
- `/mcp`

## User and Record Endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| `POST` | `/user/token/add` | add available token quota for a user |
| `GET` | `/user/list` | paginated user list |
| `DELETE` | `/user/delete?user_id=...` | delete one user |
| `POST` | `/user/insert/record` | insert user records in bulk |
| `GET` | `/record/list` | paginated conversation records |
| `DELETE` | `/record/delete?record_id=...` | delete one record |

### `POST /user/token/add`

Request body:

```json
{
  "user_id": "user123",
  "token": 100
}
```

### `GET /user/list`

Query parameters:

| Parameter | Required | Notes |
|---|---|---|
| `page` | no | default handled by server/db |
| `page_size` | no | default handled by server/db |
| `user_id` | no | optional filter |

### `GET /record/list`

Query parameters:

| Parameter | Required | Notes |
|---|---|---|
| `page` | no | default `1` |
| `page_size` | no | default `10` |
| `is_deleted` | no | `0`, `1`, or omitted |
| `user_id` | no | optional filter |
| `record_type` | no | optional filter |

## Agent Run Endpoints

These power the Admin `#/runs` page.

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/run/list` | paginated run list |
| `GET` | `/run/get?id=...` | get one run with step detail |
| `POST` | `/run/replay` | replay one historical run |
| `DELETE` | `/run/delete` | delete one run and its steps |

### `GET /run/list`

Query parameters:

| Parameter | Required | Notes |
|---|---|---|
| `page` | no | default `1` |
| `page_size` | no | default `10`; `pageSize` is also accepted |
| `mode` | no | `task`, `mcp`, or `skill` |
| `status` | no | `running`, `succeeded`, `failed` |
| `user_id` | no | optional filter; `userId` is also accepted |

### `POST /run/replay`

Accepted form/query parameter:

| Parameter | Required | Notes |
|---|---|---|
| `id` | yes | run ID to replay |

### `DELETE /run/delete`

Accepted form/query parameter:

| Parameter | Required | Notes |
|---|---|---|
| `id` | yes | preferred run ID field |
| `run_id` | no | also accepted for admin compatibility |

## Skills Endpoints

These power the Admin `#/skills` page.

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/skills/list` | list current skill catalog |
| `GET` | `/skills/detail?id=...` | get one skill detail |
| `POST` | `/skills/reload` | reload the skill catalog |
| `GET` | `/skills/validate` | validate current catalog and summarize warnings |

Notes:

- the catalog contains `local`, `builtin`, and `legacy` sources
- local skills come from `skills/*/SKILL.md`
- validation includes source counts and warning messages

## MCP Endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/mcp/get` | read current MCP config file |
| `GET` or `POST` | `/mcp/inspect` | inspect MCP config availability and setup warnings |
| `POST` | `/mcp/update?name=...` | upsert one MCP server entry |
| `DELETE` | `/mcp/delete?name=...` | delete one MCP server entry |
| `POST` | `/mcp/disable?name=...&disable=0|1` | enable or disable one MCP server |
| `POST` | `/mcp/sync` | clear clients and reinitialize MCP registrations |

### `POST /mcp/update?name=...`

Request body is one `mcpParam.MCPConfig` object, for example:

```json
{
  "url": "http://playwright-mcp:8931/mcp",
  "description": "Browser automation and inspection."
}
```

### `GET|POST /mcp/inspect`

- `GET` inspects the currently saved config
- `POST` inspects the provided config body without requiring you to save it first

The response includes:

- raw `mcpServers`
- per-server availability status
- setup, runtime, or secret warnings

## Runtime / Knowledge Endpoints

The current unified runtime and knowledge HTTP endpoints are:

| Method | Endpoint | Purpose |
|---|---|---|
| `POST` | `/runs` | create a unified chat / task / skill / workflow run |
| `GET` | `/runs/{id}` | fetch one run result |
| `GET` | `/tools/effective` | inspect the effective runtime tool set |
| `GET` | `/skills/status` | inspect skill status |
| `GET` | `/memory/status` | inspect memory status |
| `GET` | `/knowledge/status` | inspect knowledge status |
| `POST` | `/knowledge/search` | run unified knowledge retrieval |
| `POST` | `/knowledge/ingest` | ingest text or files into the unified knowledge store |

The knowledge management endpoints are:

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/knowledge/files/list` | unified knowledge file list |
| `POST` | `/knowledge/files/create` | create a unified knowledge file |
| `GET` | `/knowledge/files/get` | fetch a unified knowledge file |
| `DELETE` | `/knowledge/files/delete` | delete a unified knowledge file |
| `POST` | `/knowledge/clear` | clear unified knowledge data |
| `GET` | `/knowledge/collections/list` | list collections |
| `POST` | `/knowledge/collections/create` | create collection |
| `GET` | `/knowledge/documents/list` | list documents |
| `GET` | `/knowledge/documents/get` | get one document |
| `POST` | `/knowledge/documents/create` | create one text or binary document |
| `DELETE` | `/knowledge/documents/delete` | delete one document |
| `GET` | `/knowledge/jobs/list` | list ingestion jobs |
| `POST` | `/knowledge/retrieval/debug` | run retrieval debug |
| `GET` | `/knowledge/retrieval/runs/list` | list retrieval runs |
| `GET` | `/knowledge/retrieval/runs/get` | get one retrieval run |

The endpoints most commonly used by the current verification flow are:

- `/knowledge/collections/list`
- `/knowledge/documents/create`
- `/knowledge/jobs/list`
- `/knowledge/retrieval/debug`

## Cron Endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/cron/list` | paginated cron list |
| `POST` | `/cron/create` | create cron task |
| `POST` | `/cron/update` | update cron task |
| `POST` | `/cron/update_status` | enable or disable cron task |
| `DELETE` | `/cron/delete` | delete cron task |

## Platform / Misc Endpoints

The runtime also exposes:

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/image` | image serving helper |
| `POST` | `/com/wechat` | WeChat communication entry |
| `POST` | `/wechat` | WeChat bot entry |
| `POST` | `/qq` | QQ bot entry |
| `POST` | `/onebot` | OneBot entry |

## Practical Notes

- Admin uses its own `/bot/...` proxy routes, but those proxy to the bot-side endpoints documented here
- the current `scripts/verify.sh` flow directly checks `/pong`, `/metrics`, `/run/list`, and the knowledge endpoints
- the current `#/runs` page depends on `/run/list`, `/run/get`, `/run/replay`, and `/run/delete`
- the current `#/skills` page depends on `/skills/list`, `/skills/detail`, `/skills/reload`, and `/skills/validate`
