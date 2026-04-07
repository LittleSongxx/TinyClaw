# TinyClaw

TinyClaw is a Go-based AI bot project that connects chat platforms, Web APIs, and LLM capabilities through one shared bot core.

This repository has already been reworked into my own project layout. The currently recommended setup is:

- Platform: Feishu / Lark
- Model: Qwen via Aliyun Bailian
- Deployment: Docker Compose
- Database: SQLite
- Operations: built-in Admin panel

The codebase still contains other platform and model adapters, but this README now focuses on the setup that is actually maintained and validated in this repository.

## What TinyClaw Does

- Multi-platform bot adapters with a shared execution pipeline
- Text chat, context memory, and built-in commands
- Image, audio, and video related capabilities
- Web API access
- RAG, MCP / function calling, cron jobs, and metrics
- Admin service for configuration, records, users, and logs

## Recommended Stack

If you just want TinyClaw running quickly, use this combination:

- Feishu / Lark bot
- `qwen-max`
- Docker Compose
- SQLite

This is also the stack currently verified in this repository.

## Repository Layout

```text
TinyClaw/
├─ cmd/tinyclaw/          main application entrypoint
├─ admin/                 admin backend and frontend
├─ robot/                 platform adapters
├─ llm/                   model integrations
├─ conf/                  configuration definitions
├─ deploy/docker/         Docker deployment files
├─ scripts/               start/stop/build/release scripts
├─ docs/                  deployment-facing guides
├─ static/doc/            feature and adapter docs
├─ data/                  runtime data
└─ log/                   runtime logs
```

## Quick Start

1. Clone the repository

```bash
git clone https://github.com/LittleSongxx/TinyClaw.git
cd TinyClaw
```

2. Create your local runtime config

```bash
cp deploy/docker/.env.example deploy/docker/.env
```

Then edit `deploy/docker/.env` for your own platform and model credentials.

Minimal Feishu + Qwen example:

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

3. Start services

```bash
./scripts/start.sh
```

4. Check status

```bash
./scripts/status.sh
```

5. Safe stop helper

```bash
./scripts/stop.sh
```

This command no longer stops containers by default. It only reminds you that auto-start is enabled and prints the current stack state.

If you intentionally want to stop the Docker Compose stack:

```bash
./scripts/stop.sh --down
```

## Runtime Endpoints

- Bot HTTP: starts from port `36060`
- Admin: starts from port `18080`
- Health check: `/pong`
- Metrics: `/metrics`

Use `./scripts/status.sh` to see the actual resolved ports on your machine.

## Auto-start

The Docker Compose stack in [deploy/docker/docker-compose.yml](/home/song/code/Agent/TinyClaw/deploy/docker/docker-compose.yml) already uses `restart: unless-stopped` for the bundled services.

That means:

- after you run `./scripts/start.sh` once, the containers will auto-restart after the Docker daemon or host comes back
- the same behavior is inherited on other machines that deploy with this repository's Compose files

If you deploy on a regular Linux host, also make sure the Docker service itself starts on boot:

```bash
sudo systemctl enable --now docker
```

If you deploy with Docker Desktop, Docker Desktop startup behavior controls whether the daemon is available after login or reboot.

## Build

Build the main application:

```bash
go build ./cmd/tinyclaw
```

Build the admin service:

```bash
go build -o /tmp/TinyClawAdmin ./admin
```

Or use Makefile targets:

```bash
make build
make build-admin
make test
```

## Docs

- Deployment guide: [docs/USER_GUIDE_ZH.md](docs/USER_GUIDE_ZH.md)
- Feishu / Lark: [static/doc/lark.md](static/doc/lark.md)
- Web API: [static/doc/web_api.md](static/doc/web_api.md)
- Admin: [static/doc/admin.md](static/doc/admin.md)
- RAG: [static/doc/rag.md](static/doc/rag.md)
- Parameters: [static/doc/param_conf.md](static/doc/param_conf.md)

## Notes

- This repository has already been migrated away from the original upstream fork path into my own repo and dependency layout.
- The README now keeps only the high-signal, maintained, project-specific content.
- Adapter-specific docs are still preserved under `static/doc/` and can be trimmed further over time.
