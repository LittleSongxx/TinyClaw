# TinyClaw PC Node 接入说明

这份文档专门说明 `tinyclaw-node` 怎么和 Gateway 配对，以及它现在能做什么。

## 1. 设计目标

`tinyclaw-node` 是 TinyClaw 的真实设备执行层。

它的职责不是替代主服务，而是把某一台 PC 的能力注册到 Gateway，让 Agent 可以通过统一协议调用这台机器。

一期只关注 PC：

- Windows
- macOS
- Linux

暂时不覆盖 Android / iPhone。

## 2. 为什么节点要跑在真实主机上

因为你真正想做的这些能力：

- 截图当前桌面
- 打开浏览器或本机应用
- 后续发送键盘与鼠标事件

都天然依赖真实桌面会话。

所以推荐：

- `app` 跑在 Docker Compose
- `tinyclaw-node` 跑在真实主机

## 3. 准备条件

### Gateway 侧

在主服务环境里准备：

```env
NODE_PAIRING_TOKEN=replace-with-a-strong-node-token
```

必要时也配：

```env
GATEWAY_SHARED_SECRET=replace-with-a-strong-secret
```

### 节点侧

节点需要：

- Go 1.24+
- 能访问 Gateway 的网络
- 与 Gateway 相同的 `NODE_PAIRING_TOKEN`

## 4. 启动方式

### 方式一：直接运行

Linux / macOS:

```bash
export NODE_PAIRING_TOKEN=replace-with-a-strong-node-token
go run ./cmd/tinyclaw-node \
  --gateway_ws ws://127.0.0.1:36060/gateway/nodes/ws \
  --node_token "$NODE_PAIRING_TOKEN"
```

Windows PowerShell:

```powershell
$env:NODE_PAIRING_TOKEN="replace-with-a-strong-node-token"
go run ./cmd/tinyclaw-node --gateway_ws ws://127.0.0.1:36060/gateway/nodes/ws --node_token $env:NODE_PAIRING_TOKEN
```

### 方式二：先构建再运行

```bash
go build -o tinyclaw-node ./cmd/tinyclaw-node
./tinyclaw-node --gateway_ws ws://127.0.0.1:36060/gateway/nodes/ws --node_token "$NODE_PAIRING_TOKEN"
```

## 5. 当前支持的 capability

### `system.exec`

在节点上执行命令。

### `fs.list`

列出目录内容。

### `fs.read`

读取文件内容；文本直接返回，二进制会返回 base64。

### `fs.write`

写文件，支持覆盖与追加。

### `screen.snapshot`

抓取当前桌面截图。

### `browser.open`

用默认浏览器打开 URL。

### `app.launch`

启动本机应用或命令。

## 6. 平台实现说明

### Windows

当前截图通过 PowerShell + .NET 屏幕 API 实现。

### macOS

当前截图优先走 `screencapture`。

### Linux

当前截图依赖以下工具之一：

- `gnome-screenshot`
- `grim`
- `scrot`
- `import`

如果都没有，`screen.snapshot` 会失败。

## 7. 如何确认节点已经接入

主服务启动后，可以查询：

```bash
curl http://127.0.0.1:36060/gateway/nodes/list
```

如果节点在线，返回里会看到：

- `id`
- `name`
- `platform`
- `capabilities`

## 8. 一个最小测试

可以直接通过 Gateway 下发一个只读命令：

```bash
curl -X POST http://127.0.0.1:36060/gateway/node/command \
  -H 'Content-Type: application/json' \
  -d '{
    "capability": "fs.list",
    "arguments": {
      "path": "."
    }
  }'
```

如果当前只有一个已连接节点，会自动路由到这台节点。

## 9. 安全建议

- 为 `NODE_PAIRING_TOKEN` 使用强随机值
- 不要把 Gateway 裸露到公网
- 尽量让节点只连接到你自己控制的 Gateway
- 先从只读能力开始测试
- 真要开放写操作或命令执行时，建议配合独立审批与最小权限环境

## 10. 当前限制

这次提交已经把 Node 网络、协议和首批 PC capability 落下来了，但还不是最终形态。

当前限制包括：

- 还没有内建审批 UI
- 还没有统一的窗口管理抽象
- 还没有键盘鼠标输入能力
- 浏览器能力当前以“打开网页”为主，复杂页面自动化仍建议走 Playwright MCP

## 11. 后续扩展方向

下一批最自然的演进是：

- `input.keyboard`
- `input.mouse`
- 窗口定位 / 聚焦
- 更完整的浏览器与页面交互抽象
- Admin 直接展示 nodes / sessions / approvals 状态
