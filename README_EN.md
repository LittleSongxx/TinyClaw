# TinyClaw

Chinese documentation is now the primary entry.

- Main Chinese README: [`README_ZH.md`](README_ZH.md)
- Chinese operations guide: [`docs/USER_GUIDE_ZH.md`](docs/USER_GUIDE_ZH.md)
- PC node guide: [`docs/PC_NODE_ZH.md`](docs/PC_NODE_ZH.md)

## Overview

TinyClaw is a Go-native AI Agent / Bot platform with a new layered runtime:

- `Gateway` for auth, routing, sessions, and node ingress
- `Session` for transcript-backed conversation state
- `Agent` for reasoning and orchestration
- `Tool` for host / MCP / knowledge capabilities
- `Node` for real PC control through `tinyclaw-node`

The recommended deployment model is:

- run the main TinyClaw stack with Docker Compose
- run `tinyclaw-node` on a real Windows / macOS / Linux machine
- if you need Windows desktop control plus Linux workspace execution, install `tinyclaw-node` on Windows and enable WSL virtual nodes from the settings UI

Do not rely on a regular containerized node if you want real desktop control.

Current node support includes:

- generic PC actions such as `system.exec`, `fs.list`, `fs.read`, `fs.write`, `screen.snapshot`, `browser.open`, and `app.launch`
- Windows desktop automation for keyboard, mouse, window, and UI actions
- WSL virtual nodes with `wsl.exec`, `wsl.fs.list`, `wsl.fs.read`, and `wsl.fs.write`

## Quick Start

```bash
cp deploy/docker/.env.example deploy/docker/.env
./scripts/start.sh
```

For Windows, build and install the setup package:

```bash
./scripts/package_tinyclaw_node_windows.sh amd64
```

Then point `TinyClaw Node Settings` at `ws://127.0.0.1:36060/gateway/nodes/ws`.

For Linux or macOS:

```bash
go run ./cmd/tinyclaw-node --gateway_ws ws://127.0.0.1:36060/gateway/nodes/ws --node_token "$NODE_PAIRING_TOKEN"
```

## Key Endpoints

- `GET /pong`
- `GET /metrics`
- `WS /gateway/ws`
- `WS /gateway/nodes/ws`
- `GET /gateway/nodes/list`
- `GET /gateway/sessions/list`
- `POST /gateway/node/command`
