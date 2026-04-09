# TinyClaw PC Node 接入说明

这份文档说明 `tinyclaw-node` 如何和 Gateway 配对，以及当前这套 Windows 桌面节点 + WSL 虚拟节点是怎么工作的。

## 1. 当前运行模型

`tinyclaw-node` 是 TinyClaw 的真实设备执行层，不是主服务的替代品。

当前推荐拓扑是：

- `Docker Compose` 运行 TinyClaw 主服务、Admin、RAG 和基础依赖
- `tinyclaw-node` 跑在真实主机上，向 Gateway 注册设备能力

如果你的目标是“控制这台 Windows 电脑，同时还能执行同机 WSL 里的 Linux 命令”，推荐做法是：

- 把 `tinyclaw-node` 安装在 Windows 上
- 在设置界面里启用 Windows desktop node
- 按需启用一个或多个 WSL distro，作为独立虚拟节点暴露给 Agent

## 2. 为什么节点必须跑在真实主机上

这些能力都依赖真实交互式桌面会话：

- 截图当前桌面
- 聚焦窗口
- 发送键盘和鼠标事件
- 检查当前 UI 元素
- 打开本机浏览器或桌面应用

所以：

- 容器适合跑主服务和协议联调
- 真实桌面控制必须落在真实主机

## 3. 准备条件

### 主服务侧

至少准备：

```env
NODE_PAIRING_TOKEN=replace-with-a-strong-node-token
```

建议同时配置：

```env
GATEWAY_SHARED_SECRET=replace-with-a-strong-secret
PRIVILEGED_USER_IDS=your-feishu-or-platform-user-id
```

其中：

- `NODE_PAIRING_TOKEN` 用于节点配对
- `PRIVILEGED_USER_IDS` 可让可信管理员跳过高风险设备审批

### 节点侧

节点需要：

- 能访问 Gateway 的网络
- 与主服务相同的 `NODE_PAIRING_TOKEN`
- Windows 桌面节点场景下，保持交互式登录会话

## 4. Windows 节点推荐安装方式

### 方式一：使用安装器

在仓库根目录构建安装包：

```bash
./scripts/package_tinyclaw_node_windows.sh amd64
```

产物默认在：

- `build/release/TinyClawNodeSetup.exe`
- `build/release/TinyClawNode-windows-amd64.zip`

普通用户推荐直接使用 `TinyClawNodeSetup.exe`。

安装完成后，会生成：

- `TinyClaw Node` 桌面快捷方式
- `TinyClaw Node Settings` 桌面快捷方式
- `%ProgramData%\TinyClawNode\config.json`
- `%ProgramData%\TinyClawNode\logs`

### 方式二：便携包

如果你不想走安装器，也可以解压 zip 后双击：

- `install-node.cmd`
- `TinyClaw Node Settings`

不再要求手动输入 PowerShell 命令。

## 5. Windows 设置界面里该配什么

设置界面会写入 `%ProgramData%\TinyClawNode\config.json`。当前关键字段是：

- `gateway_ws`
- `node_token`
- `node_id`
- `node_name`
- `log_dir`
- `start_at_login`
- `enable_windows_node`
- `wsl_distros`

`wsl_distros` 每项支持：

- `name`
- `enabled`
- `allow_command_prefixes`
- `allow_write_path_prefixes`
- `default_cwd`

一个典型示例：

```json
{
  "gateway_ws": "ws://127.0.0.1:36060/gateway/nodes/ws",
  "node_token": "",
  "node_id": "DESKTOP-1234",
  "node_name": "DESKTOP-1234",
  "log_dir": "C:\\ProgramData\\TinyClawNode\\logs",
  "start_at_login": false,
  "enable_windows_node": true,
  "wsl_distros": [
    {
      "name": "Ubuntu-22.04",
      "enabled": true,
      "allow_command_prefixes": [
        "git status",
        "go test"
      ],
      "allow_write_path_prefixes": [
        "/home/user/workspace"
      ],
      "default_cwd": "/home/user/workspace/project"
    }
  ]
}
```

当前推荐最小配置是：

- 启用 Windows desktop node
- 只启用你真正要暴露的 WSL distro
- 为常用 Linux 仓库设置 `default_cwd`

## 6. Linux / macOS 开发调试方式

如果你只是做协议或通用能力联调，也可以直接运行：

```bash
export NODE_PAIRING_TOKEN=replace-with-a-strong-node-token
go run ./cmd/tinyclaw-node \
  --gateway_ws ws://127.0.0.1:36060/gateway/nodes/ws \
  --node_token "$NODE_PAIRING_TOKEN"
```

Windows 上开发调试也可以直接运行：

```powershell
tinyclaw-node.exe --config "$env:ProgramData\TinyClawNode\config.json"
```

或者打开设置界面：

```powershell
tinyclaw-node.exe --configure --config "$env:ProgramData\TinyClawNode\config.json"
```

## 7. 当前支持的 capability

### 通用 PC 节点能力

- `system.exec`
- `fs.list`
- `fs.read`
- `fs.write`
- `screen.snapshot`
- `browser.open`
- `app.launch`

### Windows 桌面自动化能力

这部分当前面向 Windows 桌面会话，支持：

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

当前这套桌面自动化可在两类运行时里使用：

- `tinyclaw-node` 直接跑在 Windows 上
- 服务跑在 WSL，但可以桥接到同机 Windows 桌面

### WSL 虚拟节点能力

每个启用的 distro 都会注册成一个独立节点，提供：

- `wsl.exec`
- `wsl.fs.list`
- `wsl.fs.read`
- `wsl.fs.write`

这意味着 Agent 可以：

- 在 Windows 节点上做截图、窗口和键鼠操作
- 在 WSL 节点上做 Git、构建、包管理和 Linux 文件操作

## 8. 审批与安全

当前默认审批规则是：

- `input.*` 这类桌面输入操作默认需要审批
- `wsl.exec` 默认需要审批
- `wsl.fs.write` 默认需要审批

当前支持三种放行方式：

- 在聊天里回复“确认”或“取消”
- 使用 `/approve <approval_id>` 或 `/reject <approval_id>`
- 把可信管理员加入 `PRIVILEGED_USER_IDS`

WSL 还支持按前缀白名单跳过审批：

- `allow_command_prefixes`
- `allow_write_path_prefixes`

建议：

- 只给受信任用户加入 `PRIVILEGED_USER_IDS`
- 只对白名单放行高频、安全、可预测的命令
- 不要把 Gateway 直接裸露到公网

## 9. 如何确认节点已经接入

### 方法一：HTTP 查询

```bash
curl http://127.0.0.1:36060/gateway/nodes/list
```

你应该能看到：

- 一个 `kind=windows` 的节点
- 一个或多个 `kind=wsl` 的节点

### 方法二：Admin

打开：

```text
http://127.0.0.1:18080
```

进入 `Nodes` 页面，检查：

- 节点在线状态
- `platform`
- `metadata.kind`
- `metadata.wsl_distro`
- 当前待审批列表

### 方法三：直接在飞书等聊天入口验证

先测 Windows：

- “在 Windows 上打开记事本并输入 hello”
- “截图当前窗口”

再测 WSL：

- “在 WSL Ubuntu-22.04 里执行 pwd && git status”
- “在 WSL Ubuntu-22.04 里读取 /home/.../README_ZH.md”

## 10. 常见排查

### 节点连不上 Gateway

优先检查：

- `NODE_PAIRING_TOKEN` 是否一致
- `gateway_ws` 是否正确
- `http://127.0.0.1:36060/pong` 是否可访问
- `/gateway/nodes/ws` 是否被代理或防火墙拦截

### Windows 节点在线，但桌面操作失败

通常是这些原因：

- 当前没有活动桌面会话
- 程序跑在了不具备桌面上下文的环境里
- 目标窗口没有聚焦
- UI 元素定位条件过严

### WSL 节点没有出现

优先检查：

- distro 名称是否和 `wsl -l -q` 一致
- 设置界面里是否勾选了该 distro
- `default_cwd` 是否是合法 Linux 路径
- 日志里是否出现 “configured wsl distro is unavailable, skipping”

## 11. 当前限制

- 更完整的浏览器 DOM 自动化仍建议走 Playwright MCP
- `system.exec` / `fs.*` 仍按节点所在环境语义执行
- WSL 当前是一次性命令执行，不是长会话 shell
- GUI 设置界面会保留 WSL allowlist，但不直接编辑它们

## 12. 延伸文档

- 项目总览：[`../README_ZH.md`](../README_ZH.md)
- 使用与运维：[`./USER_GUIDE_ZH.md`](./USER_GUIDE_ZH.md)
- Windows 安装包说明：[`../deploy/windows-node/README.md`](../deploy/windows-node/README.md)
