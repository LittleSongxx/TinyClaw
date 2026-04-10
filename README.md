# TinyClaw

中文文档现在是主入口。

- 完整中文说明：[`README_ZH.md`](README_ZH.md)
- 中文部署与运维手册：[`docs/USER_GUIDE_ZH.md`](docs/USER_GUIDE_ZH.md)
- PC 节点接入说明：[`docs/PC_NODE_ZH.md`](docs/PC_NODE_ZH.md)
- English overview: [`README_EN.md`](README_EN.md)

## 项目定位

TinyClaw 是一个基于 Go 的 AI Agent / Bot 平台。当前主线架构不再只是“多平台聊天机器人”，而是拆成了几层清晰职责：

- `Gateway`：鉴权、路由、控制面、节点接入
- `Session`：会话上下文、transcript、会话索引
- `Agent`：推理、决策、调度
- `Tool`：本机工具、MCP 工具、知识工具
- `Node`：真实 PC 设备能力，负责远程执行和桌面操作

这次重构的重点是：

- 保持 Go 原生主干
- 不照搬 openclaw，只借鉴分层设计
- 一期优先支持 PC，而不是手机
- 让 TinyClaw 能通过 `tinyclaw-node` 去操作真实电脑

## 当前推荐部署方式

- `Docker Compose` 负责 TinyClaw 主服务与依赖
- `tinyclaw-node` 跑在真实 Windows / macOS / Linux 主机上
- 如果你要同时控制 Windows 桌面和 WSL 工作区，推荐把 `tinyclaw-node` 安装在 Windows 上，再在设置界面里开启一个或多个 WSL 虚拟节点
- `Gateway` 仍然由主 `app` 服务提供
- `Session transcript` 默认落在 `data/sessions/`

如果你的目标是“控制真实桌面”，不要把 `tinyclaw-node` 跑在普通容器里替代真实主机。容器里的节点适合协议联调，不适合桌面自动化。

当前节点能力包括：

- 通用 PC 能力：`system.exec`、`fs.list`、`fs.read`、`fs.write`、`screen.snapshot`、`browser.open`、`app.launch`
- Windows 桌面自动化：键盘、鼠标、窗口和 UI inspect/find/focus
- WSL 虚拟节点：`wsl.exec`、`wsl.fs.list`、`wsl.fs.read`、`wsl.fs.write`

## 快速开始

1. 准备配置

```bash
cp deploy/docker/.env.example deploy/docker/.env
```

至少补齐这几类配置：

- 平台与模型凭据，如 `LARK_APP_ID`、`LARK_APP_SECRET`、`ALIYUN_TOKEN`
- Gateway 安全参数，如 `GATEWAY_SHARED_SECRET`
- Node 通过 `devices.bootstrap` 生成短期 pairing code，审批后写入 `device_token`
- Workspace 管理员由默认 workspace membership 和 actor token 决定
- 如果启用默认 MCP，再补 `AMAP_MAPS_API_KEY`、`BOCHA_API_KEY`、`GITHUB_PERSONAL_ACCESS_TOKEN`

2. 启动主服务栈

```bash
./scripts/start.sh
```

3. 检查主服务

```bash
./scripts/status.sh
./scripts/verify.sh
```

4. 在真实 PC 上启动节点

Windows 推荐方式：

```bash
./scripts/package_tinyclaw_node_windows.sh amd64
```

然后在 Windows 上安装 `build/release/TinyClawNodeSetup.exe`，通过 `TinyClaw Node Settings` 填写：

- `gateway_ws=ws://127.0.0.1:36060/gateway/nodes/ws`
- `workspace_id`、`device_id`；节点密钥会自动生成并持久化
- 初次配对填写 `pairing_code`，审批后保存 `device_token`
- 按需启用 Windows 桌面节点和 WSL distro

Linux / macOS 调试方式：

```bash
export TINYCLAW_PAIRING_CODE=pair_10_minute_code
go run ./cmd/tinyclaw-node \
  --gateway_ws ws://127.0.0.1:36060/gateway/nodes/ws \
  --workspace_id default \
  --device_id "$(hostname)" \
  --pairing_code "$TINYCLAW_PAIRING_CODE"
```

如果 Gateway 不在本机，把地址替换成实际可访问的主服务地址。

## 关键目录

```text
TinyClaw/
├─ cmd/tinyclaw/          主程序入口
├─ cmd/tinyclaw-node/     PC 节点守护进程
├─ gateway/               Gateway 控制面与 WS 协议
├─ session/               会话与 transcript 存储
├─ node/                  节点协议、能力与本地驱动
├─ agent/                 Agent 运行时抽象
├─ tooling/               Tool registry / broker
├─ robot/                 现有平台接入层
├─ http/                  HTTP API 与 Gateway 入口
├─ admin/                 管理后台
├─ deploy/docker/         Docker 部署文件
├─ deploy/windows-node/   Windows 节点安装器与配置脚本
├─ docs/                  当前主线中文文档
└─ skills/                本地 SKILL.md 定义
```

## 当前新增接口

- `GET /pong`
- `GET /metrics`
- `WS /gateway/ws`
- `WS /gateway/nodes/ws`
- `GET /gateway/nodes/list`
- `GET /gateway/sessions/list`
- `POST /gateway/node/command`

## 文档导航

- 项目总览：[`README_ZH.md`](README_ZH.md)
- 使用与运维：[`docs/USER_GUIDE_ZH.md`](docs/USER_GUIDE_ZH.md)
- PC Node：[`docs/PC_NODE_ZH.md`](docs/PC_NODE_ZH.md)
