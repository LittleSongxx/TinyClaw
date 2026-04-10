# TinyClaw

TinyClaw 现在的主线定位，是一个基于 Go 的 AI Agent / Bot 平台，而不是单纯的聊天机器人集合。

当前默认运行形态已经收紧为 **Agent 平台核心 profile**：

- 默认启用：Gateway、Session、Agent Runtime、MCP/Skill、Node、Approval、Runs/Admin
- 默认保留核心入口：Web 与 Lark
- 默认关闭可选模块：Knowledge、媒体生成、Cron、legacy bot adapters、legacy MCP proxy、legacy task tools
- 默认关闭实验模块：workflow mode

这一轮重构之后，项目的主干已经切成了：

- `Gateway`：鉴权、路由、控制面、节点接入
- `Session`：会话 ID、上下文、transcript、索引
- `Agent`：推理、决策、调度
- `Tool`：本机能力、MCP、知识检索
- `Node`：真实 PC 设备执行层

它借鉴的是分层设计思路，而不是照搬别人的代码或产品包装。

## 当前项目重点

当前仓库已经开始围绕新的完整架构演进，最重要的变化有：

- 主程序仍然坚持 Go 原生实现
- `Gateway` 已经接入现有 `Web` 与 `Lark` 入口
- `Session` 已替代“只按 user_id 拼上下文”的单一模型
- `tinyclaw-node` 已经可以主动连接 Gateway，并按配置同时注册 Windows 节点和 WSL 虚拟节点
- 当前通用 PC 节点支持：
  - `system.exec`
  - `fs.list`
  - `fs.read`
  - `fs.write`
  - `screen.snapshot`
  - `browser.open`
  - `app.launch`
- 当前 Windows 桌面自动化支持：
  - `input.keyboard.*`
  - `input.mouse.*`
  - `window.list`
  - `window.focus`
  - `ui.inspect`
  - `ui.find`
  - `ui.focus`
- 当前 WSL 虚拟节点支持：
  - `wsl.exec`
  - `wsl.fs.list`
  - `wsl.fs.read`
  - `wsl.fs.write`
- Windows 侧已经提供正式安装包 `TinyClawNodeSetup.exe`，普通用户不再需要手动敲 PowerShell 才能完成安装和配置

## 推荐部署方式

当前推荐把部署拆成两块：

1. `Docker Compose`
作用：运行 TinyClaw 主服务、Admin、MCP 相关基础设施。Knowledge 依赖现在是可选 profile。

2. `tinyclaw-node`
作用：运行在真实 Windows / macOS / Linux 主机上，向 Gateway 注册自身能力。

这个边界很重要：

- 如果你只是想跑 Agent 平台、MCP、Admin，Compose 默认栈就够了
- 如果你要启用 Knowledge，请显式设置 `ENABLE_KNOWLEDGE=true` 并启动 `knowledge` profile
- 如果你想操作真实桌面，不要把 `tinyclaw-node` 当成普通容器服务来替代宿主机

## 目录结构

```text
TinyClaw/
├─ cmd/tinyclaw/          主程序入口
├─ cmd/tinyclaw-node/     PC 节点守护进程
├─ gateway/               Gateway 控制面与 WS 协议
├─ session/               会话 transcript 存储与索引
├─ node/                  节点协议、管理器、本地驱动
├─ agent/                 Agent 运行时抽象
├─ tooling/               Tool registry / broker
├─ robot/                 现有聊天平台接入层
├─ http/                  HTTP API 与 Gateway 接入点
├─ admin/                 管理后台
├─ deploy/docker/         Docker 部署文件
├─ deploy/windows-node/   Windows 节点安装器、配置脚本与 NSIS 配置
├─ docs/                  中文项目文档
├─ data/                  运行数据与 session transcript
└─ log/                   日志
```

## 快速开始

### 1. 克隆仓库

```bash
git clone https://github.com/LittleSongxx/TinyClaw.git
cd TinyClaw
```

### 2. 准备配置

```bash
cp deploy/docker/.env.example deploy/docker/.env
```

你现在至少要关注这几类参数：

- 平台凭据：`LARK_APP_ID`、`LARK_APP_SECRET`
- 模型凭据：`ALIYUN_TOKEN`
- Gateway 安全：`GATEWAY_SHARED_SECRET`
- Node 配对：`NODE_PAIRING_TOKEN`
- 可信管理员直通设备操作：`PRIVILEGED_USER_IDS`
- Session transcript 路径：`SESSION_TRANSCRIPT_DIR`

一个最小示例：

```env
BOT_NAME=TinyClawLark
LANG=zh
TYPE=aliyun
DEFAULT_MODEL=qwen-max
DB_TYPE=sqlite3
ENABLE_KNOWLEDGE=false
ENABLE_MEDIA=false
ENABLE_CRON=false
ENABLE_LEGACY_BOTS=false
ENABLE_LEGACY_MCP_PROXY=false
ENABLE_LEGACY_TASK_TOOLS=false
ENABLE_EXPERIMENTAL_WORKFLOW=false

LARK_APP_ID=your_lark_app_id
LARK_APP_SECRET=your_lark_app_secret
ALIYUN_TOKEN=your_qwen_api_key

GATEWAY_SHARED_SECRET=replace-with-a-strong-secret
NODE_PAIRING_TOKEN=replace-with-a-strong-node-token
SESSION_TRANSCRIPT_DIR=/app/data/sessions
```

如果你启用仓库默认的联网 / 浏览器 / GitHub MCP，还要补：

```env
AMAP_MAPS_API_KEY=
BOCHA_API_KEY=
GITHUB_PERSONAL_ACCESS_TOKEN=
```

### 3. 启动主服务

```bash
./scripts/start.sh
```

### 4. 检查状态

```bash
./scripts/status.sh
./scripts/verify.sh
```

如果你要做更完整的在线验证：

```bash
./scripts/verify.sh --full
```

### 5. 在真实 PC 上启动节点

Windows 推荐方式：

```bash
./scripts/package_tinyclaw_node_windows.sh amd64
```

然后在 Windows 上安装 `build/release/TinyClawNodeSetup.exe`，通过 `TinyClaw Node Settings` 填写：

- `gateway_ws=ws://127.0.0.1:36060/gateway/nodes/ws`
- `node_token` 与主服务侧 `NODE_PAIRING_TOKEN` 保持一致
- 勾选 Windows desktop node
- 按需启用 `Ubuntu-22.04` 等 WSL distro，并填写 `default_cwd`

Linux / macOS 开发调试：

```bash
export NODE_PAIRING_TOKEN=replace-with-the-same-token
go run ./cmd/tinyclaw-node \
  --gateway_ws ws://127.0.0.1:36060/gateway/nodes/ws \
  --node_token "$NODE_PAIRING_TOKEN"
```

## 当前新增接口

除了已有的 Bot API 与 Admin 入口，现在还新增了 Gateway / Node 相关接口：

- `GET /pong`
- `GET /metrics`
- `WS /gateway/ws`
- `WS /gateway/nodes/ws`
- `GET /gateway/nodes/list`
- `GET /gateway/sessions/list`
- `POST /gateway/node/command`

其中：

- `/gateway/ws` 主要给控制面或管理端使用
- `/gateway/nodes/ws` 给 `tinyclaw-node` 配对与长连接
- `/gateway/node/command` 给控制面下发节点命令

## 默认与可选能力

当前仓库里的 [deploy/docker/docker-compose.yml](deploy/docker/docker-compose.yml) 默认负责：

- `app`
- `playwright-mcp`

Knowledge 相关服务已经移动到 `knowledge` profile：

- `hf-embeddings`
- `postgres`
- `redis`
- `minio`

完整旧式向量栈仍保留在 `full` profile：

- `milvus`
- `etcd`

它不会自动替你运行一个“真实桌面节点”。

原因很简单：

- 真正的屏幕截图、应用启动、桌面输入通常需要真实用户环境
- 常规容器不等于宿主机桌面

所以文档已经明确改成：

- Compose 跑服务栈
- `tinyclaw-node` 跑在真实 PC 上

## 文档导航

- 项目收紧说明：[docs/PROJECT_FOCUS_ZH.md](docs/PROJECT_FOCUS_ZH.md)
- 部署与运维手册：[docs/USER_GUIDE_ZH.md](docs/USER_GUIDE_ZH.md)
- PC Node 说明：[docs/PC_NODE_ZH.md](docs/PC_NODE_ZH.md)
- 飞书说明：[static/doc/lark_ZH.md](static/doc/lark_ZH.md)
- Web API 说明：[static/doc/web_api_ZH.md](static/doc/web_api_ZH.md)
- Admin 说明：[static/doc/admin_ZH.md](static/doc/admin_ZH.md)
- MCP / Skills：[static/doc/functioncall_ZH.md](static/doc/functioncall_ZH.md)
- Knowledge 说明：[static/doc/knowledge_ZH.md](static/doc/knowledge_ZH.md)
- 参数说明：[static/doc/param_conf_ZH.md](static/doc/param_conf_ZH.md)

## 开发与构建

构建主程序：

```bash
go build ./cmd/tinyclaw
```

构建 PC 节点：

```bash
go build ./cmd/tinyclaw-node
```

构建 Admin：

```bash
go build -o /tmp/TinyClawAdmin ./admin
```

或直接：

```bash
make build
make build-admin
make test
```
