# TinyClaw Admin

`TinyClaw Admin` is the built-in management panel for TinyClaw. It is used to inspect runtime status, manage bot configuration, review chat records, inspect agent runs, and operate Skills, RAG, MCP, users, and cron jobs.

This document now describes the current TinyClaw admin usage instead of the old single-platform bot wording.

## Recommended Usage

If you are following the current repository layout, the recommended way is to start the full stack with Docker Compose:

```bash
./scripts/start.sh
```

The admin service starts together with the main TinyClaw service.

The internal admin port is `18080`. The actual host port depends on runtime port resolution, so check it with:

```bash
./scripts/status.sh
```

## Default Login

On first initialization, the default credentials are:

- Username: `admin`
- Password: `admin`

Change the password immediately after the first login.

## Main Pages

The admin panel is most useful for these areas:

- `Dashboard`
  runtime overview and basic metrics
- `Runs`
  agent run traces, step details, tool observations, replay, single delete, and batch delete
- `Bots`
  bot configuration management
- `BotUsers`
  user records, modes, and token limits
- `BotChats`
  chat history inspection
- `Chat`
  direct admin-side bot debugging
- `Log`
  runtime logs
- `RAG`
  knowledge management through `Documents / Ingestion Jobs / Retrieval Debug`
- `MCP`
  MCP service configuration, prepared templates, and availability inspection
- `Skills`
  skill catalog management with local / builtin / legacy grouping, validation, reload, and detail view
- `Cron`
  scheduled task management

## How It Runs

### Option 1: Docker Compose

This is the default project workflow.

```bash
./scripts/start.sh
./scripts/status.sh
./scripts/verify.sh
./scripts/stop.sh
```

`./scripts/stop.sh` is now a safe helper and does not stop containers by default.
Use `./scripts/stop.sh --down` only when you intentionally want to stop the Compose stack.

If you want a full live validation for the Agent and RAG stack, run:

```bash
./scripts/verify.sh --full
```

### Option 2: Run Admin Separately

If you want to debug the admin service by itself:

```bash
go build -o /tmp/TinyClawAdmin ./admin
```

Then provide the required environment variables manually:

- `DB_TYPE`
- `DB_CONF`
- `SESSION_KEY`
- `ADMIN_PORT`

Example:

```bash
DB_TYPE=sqlite3 \
DB_CONF=./data/tiny_claw_admin.db \
SESSION_KEY=replace-with-your-session-key \
ADMIN_PORT=18080 \
/tmp/TinyClawAdmin
```

## Key Variables

| Variable | Purpose | Example |
|---|---|---|
| `DB_TYPE` | admin database type | `sqlite3` |
| `DB_CONF` | admin database file or DSN | `./data/tiny_claw_admin.db` |
| `SESSION_KEY` | session signing key | random long string |
| `ADMIN_PORT` | listen port | `18080` |

## Relation to the Main Service

`TinyClaw Admin` is not a separate product. It works alongside the main TinyClaw runtime and shares the same deployment model.

The most relevant files are:

- main database: `data/tiny_claw.db`
- admin database: `data/tiny_claw_admin.db`
- Agent / RAG v2 services: `postgres + redis + minio`
- default MCP config: `conf/mcp/mcp.json`
- local skills directory: `skills/`
- main log: `log/tiny_claw.log`
- runtime config: `deploy/docker/.env`

## Common Issues

### Admin page does not open

Check:

- container health
- actual mapped host port
- whether `SESSION_KEY` changed
- whether `tiny_claw_admin.db` still exists

### Session suddenly became invalid

Usually caused by:

- environment rebuild
- `SESSION_KEY` change
- stale browser cookies

Re-login is usually enough.

### Admin opens but no bot data appears

Check:

- TinyClaw main service is running
- main service is writing to `data/tiny_claw.db`
- runtime config points to the expected data directory

### MCP or Skills pages show warnings

Check:

- `conf/mcp/mcp.json` is readable and valid JSON
- required secrets such as `AMAP_MAPS_API_KEY`, `BOCHA_API_KEY`, and `GITHUB_PERSONAL_ACCESS_TOKEN` are set when those servers are enabled
- the underlying MCP commands or URLs are actually reachable from the `app` container
- local skill files under `skills/` still contain valid `SKILL.md` frontmatter and required sections
