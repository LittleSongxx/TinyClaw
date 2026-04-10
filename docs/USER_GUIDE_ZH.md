# TinyClaw 使用与运维手册

这份文档面向当前已经完成新架构升级后的 TinyClaw，重点回答三个问题：

1. 这套系统现在怎么组成
2. 日常怎么启动、检查、排障
3. 怎么把真实 PC 接进来交给 TinyClaw 控制

## 1. 当前架构

当前主线架构已经拆成：

- `Gateway`
  负责鉴权、路由、Session 接入、Node 接入、控制面接口
- `Agent`
  负责推理、决策、调度
- `Session`
  负责 transcript、上下文、会话索引
- `Tool`
  负责 MCP / 本机 / 检索类能力抽象
- `Node`
  负责远程 PC 执行能力

和旧版本最大的不同是：

- 入口不再只是直接调用 `robot -> llm`
- 现在先经过 Gateway，再进入既有执行流
- Session 已经成为独立层
- 设备控制不再混进 tool 内核，而是进入专门的 Node 网络
- Knowledge、媒体生成、Cron、legacy bot adapters、legacy MCP proxy 默认关闭，作为可选模块启用

## 2. 推荐运行方式

推荐把系统拆成两部分：

### 主服务栈

使用 Docker Compose 运行：

- TinyClaw `app`
- Admin
- Playwright MCP

Knowledge 相关的 PostgreSQL / Redis / MinIO / HuggingFace embedding 服务现在属于可选 `knowledge` profile。

### 真实 PC 节点

在真实 Windows / macOS / Linux 上运行：

- `tinyclaw-node`

这一步非常关键。

如果你的目标是：

- 截图真实桌面
- 启动本机应用
- 未来发送键盘鼠标事件

那就必须把节点跑在真实主机上，而不是仅仅跑在普通容器里。

## 3. 关键文件

- [`deploy/docker/.env.example`](../deploy/docker/.env.example)
  默认配置模板
- [`deploy/docker/docker-compose.yml`](../deploy/docker/docker-compose.yml)
  Compose 编排
- [`cmd/tinyclaw/main.go`](../cmd/tinyclaw/main.go)
  主服务入口
- [`cmd/tinyclaw-node/main.go`](../cmd/tinyclaw-node/main.go)
  PC 节点入口
- [`gateway/`](../gateway)
  Gateway 协议与服务
- [`session/`](../session)
  transcript 与会话存储
- [`node/`](../node)
  Node 管理器与设备驱动
- [`http/gateway.go`](../http/gateway.go)
  Gateway HTTP / WS 接口

## 4. 配置项变化

旧版部署主要关心平台和模型凭据。现在还需要额外关心 Gateway / Node / Session 这些参数：

```env
GATEWAY_WS_PATH=/gateway/ws
GATEWAY_NODE_WS_PATH=/gateway/nodes/ws
GATEWAY_SHARED_SECRET=replace-with-a-strong-secret
NODE_PAIRING_TOKEN=replace-with-a-strong-node-token
SESSION_TRANSCRIPT_DIR=/app/data/sessions
```

默认 profile 已经收紧为 Agent 平台核心能力。以下可选模块默认关闭：

```env
ENABLE_KNOWLEDGE=false
ENABLE_MEDIA=false
ENABLE_CRON=false
ENABLE_LEGACY_BOTS=false
ENABLE_LEGACY_MCP_PROXY=false
ENABLE_LEGACY_TASK_TOOLS=false
ENABLE_EXPERIMENTAL_WORKFLOW=false
```

启用 Knowledge 时，需要同时启动 Docker profile：

```bash
ENABLE_KNOWLEDGE=true docker compose --profile knowledge up -d
```

说明：

- `GATEWAY_SHARED_SECRET`
  用于 Gateway 控制面鉴权
- `NODE_PAIRING_TOKEN`
  用于 `tinyclaw-node` 配对
- `SESSION_TRANSCRIPT_DIR`
  用于保存 JSONL transcript

## 5. 日常启动与检查

进入仓库目录：

```bash
cd /path/to/TinyClaw
```

启动：

```bash
./scripts/start.sh
```

查看状态：

```bash
./scripts/status.sh
```

快速自检：

```bash
./scripts/verify.sh
```

完整自检：

```bash
./scripts/verify.sh --full
```

安全停止辅助：

```bash
./scripts/stop.sh
```

如果你明确要停止 Compose：

```bash
./scripts/stop.sh --down
```

## 6. Gateway 相关接口

当前新增接口如下：

- `GET /pong`
- `GET /metrics`
- `WS /gateway/ws`
- `WS /gateway/nodes/ws`
- `GET /gateway/nodes/list`
- `GET /gateway/sessions/list`
- `POST /gateway/node/command`

你可以这样理解：

- `/gateway/ws`
  面向控制面或管理端
- `/gateway/nodes/ws`
  面向节点长连接
- `/gateway/nodes/list`
  查看已注册节点
- `/gateway/sessions/list`
  查看 Session 元数据
- `/gateway/node/command`
  向节点下发命令

## 7. Session 现在怎么存

旧版更偏向“按 user_id 拼聊天记录”。

现在会话系统分成两层：

- 文件层
  JSONL transcript，默认在 `data/sessions/`
- 索引层
  数据库 `sessions` 表，记录 `session_id / session_key / channel / peer / transcript_path / message_count`

这带来的好处是：

- 更容易回放与迁移
- 群聊 / 私聊 / 调试 / 定时任务能更清晰隔离
- 未来更容易加摘要压缩与长期记忆

## 8. PC Node 如何使用

详细说明见：

- [`docs/PC_NODE_ZH.md`](./PC_NODE_ZH.md)

这里先给一个最小流程。

### 第一步：主服务已经启动

确认：

```text
http://127.0.0.1:36060/pong
```

可访问。

### 第二步：准备相同的配对 token

主服务侧：

```env
NODE_PAIRING_TOKEN=replace-with-a-strong-node-token
```

节点侧也使用同一个值。

### 第三步：在真实主机运行节点

Windows 推荐方式：

```bash
./scripts/package_tinyclaw_node_windows.sh amd64
```

然后在 Windows 上安装 `build/release/TinyClawNodeSetup.exe`，打开 `TinyClaw Node Settings`，填写：

- `gateway_ws=ws://127.0.0.1:36060/gateway/nodes/ws`
- `node_token` 与主服务保持一致
- 勾选 Windows desktop node
- 按需启用 `Ubuntu-22.04` 等 WSL distro

Linux / macOS 开发调试方式：

```bash
go run ./cmd/tinyclaw-node \
  --gateway_ws ws://127.0.0.1:36060/gateway/nodes/ws \
  --node_token "$NODE_PAIRING_TOKEN"
```

### 第四步：查询节点

```bash
curl http://127.0.0.1:36060/gateway/nodes/list
```

## 9. 当前已实现的 PC 能力

当前 `tinyclaw-node` 已实现这些 capability：

- `system.exec`
- `fs.list`
- `fs.read`
- `fs.write`
- `screen.snapshot`
- `browser.open`
- `app.launch`
- `input.keyboard.type`
- `input.keyboard.key`
- `input.keyboard.hotkey`
- `input.mouse.move`
- `input.mouse.click`
- `input.mouse.double_click`
- `input.mouse.right_click`
- `input.mouse.drag`
- `window.list`
- `window.focus`
- `ui.inspect`
- `ui.find`
- `ui.focus`

如果启用了 WSL 虚拟节点，还会额外提供：

- `wsl.exec`
- `wsl.fs.list`
- `wsl.fs.read`
- `wsl.fs.write`

当前更完整的浏览器 DOM 自动化仍建议走 Playwright MCP。

## 10. 设备控制安全建议

建议至少做到：

- `GATEWAY_SHARED_SECRET` 和 `NODE_PAIRING_TOKEN` 使用强随机值
- 可信管理员才加入 `PRIVILEGED_USER_IDS`
- 不要把 Gateway 直接裸露在公网
- 如果必须远程访问，放在 VPN / Tailscale / 内网穿透保护之后
- 先只给受信任主机配对
- 先从只读类能力开始测试，例如 `fs.read`、`screen.snapshot`
- 对 `wsl.exec` / `wsl.fs.write` 尽量使用前缀白名单，而不是完全放开

## 11. 常见排查

### 节点连不上 Gateway

优先检查：

- `NODE_PAIRING_TOKEN` 是否一致
- `gateway_ws` 地址是否正确
- 主服务端口是否映射正确
- `/gateway/nodes/ws` 是否被反向代理或防火墙拦截

### 能连上，但桌面能力不正常

通常是运行环境问题：

- 节点跑在容器里而不是真实桌面
- Linux 宿主机缺少 `gnome-screenshot` / `grim` / `scrot`
- macOS 没开辅助功能权限
- Windows 会话没有活动桌面
- WSL distro 名称配置不对，导致虚拟节点没有注册上来

### 聊天里设备操作一直要确认

优先检查：

- 当前用户是否属于 `PRIVILEGED_USER_IDS`
- 操作是否属于默认需要审批的能力，例如 `input.*`、`wsl.exec`、`wsl.fs.write`
- 飞书里是否直接回复了“确认”或“取消”
- 是否在用 `/approve <approval_id>` 或 `/reject <approval_id>`

### Session 没看到历史

优先检查：

- `SESSION_TRANSCRIPT_DIR` 是否落在持久化目录
- `data/sessions/` 是否存在 transcript
- 数据库 `sessions` 表是否正常写入

## 12. 延伸文档

- 项目总览：[`../README_ZH.md`](../README_ZH.md)
- PC Node：[`./PC_NODE_ZH.md`](./PC_NODE_ZH.md)
- 飞书：[`../static/doc/lark_ZH.md`](../static/doc/lark_ZH.md)
- Web API：[`../static/doc/web_api_ZH.md`](../static/doc/web_api_ZH.md)
