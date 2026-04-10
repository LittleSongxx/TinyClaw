# TinyClaw 项目收紧说明

TinyClaw 当前最有差异化的方向不是“更多 AI bot 功能”，而是一个可审计、可治理、能连接真实执行环境的 Agent 平台。

## 核心保留

- `Gateway`：统一接入 Web/Lark、控制面、Node 长连接。
- `Session`：隔离群聊、私聊、控制台、节点任务的上下文和 transcript。
- `Agent Runtime`：统一 chat/task/mcp/skill 的运行记录、步骤、输出和错误。
- `Tool Broker`：将 MCP、Node、未来本机能力纳入同一执行面。
- `Node`：让 Agent 可以操作真实 PC、WSL、文件、窗口、屏幕和应用。
- `Approval`：高风险操作必须可确认、可拒绝、可追踪。
- `Admin`：以 Runs、Nodes、Approvals、Skills、Tools 为主的控制台。

## 默认关闭

以下能力仍可保留，但不再属于默认产品主线：

- Knowledge/RAG：通过 `ENABLE_KNOWLEDGE=true` 启用。
- 媒体生成：通过 `ENABLE_MEDIA=true` 启用。
- Cron：通过 `ENABLE_CRON=true` 启用。
- Telegram/Discord/Slack/Ding/Wechat/QQ 等 legacy bot adapters：通过 `ENABLE_LEGACY_BOTS=true` 启用。
- 自动生成的 legacy MCP proxy skills：通过 `ENABLE_LEGACY_MCP_PROXY=true` 启用。
- 旧 MCP server-level task tools：通过 `ENABLE_LEGACY_TASK_TOOLS=true` 启用。
- 实验性 workflow mode：通过 `ENABLE_EXPERIMENTAL_WORKFLOW=true` 启用。

## 不成熟区域

- `robot/robot.go` 仍然过大，后续应拆成 channel adapter、command router、message sender、runtime bridge。
- `runtimecore`、`agentruntime`、`agent` 仍有职责重叠，后续应统一为一个公开 Runtime 入口。
- `ModeWorkflow` 当前仍接近 task 模式，已经默认关闭，避免作为成熟 workflow engine 宣传。
- Skill catalog 默认暴露面已经收紧，但后续还应为 tool 增加来源、风险、审批策略和可观测元数据。
- Knowledge 模块仍然偏重，适合继续拆成独立模块或服务，而不是默认绑定主启动路径。

## 后续重构优先级

1. 拆分 `robot` 包，让 Web/Lark 变成清晰的核心 channel adapter。
2. 统一 Runtime 生命周期，让 chat/task/mcp/skill 都写入一致的 run/step/audit。
3. 将 Node approval decision 和 grant 命中持久化，形成完整 action timeline。
4. 把 Knowledge、media、legacy bot、cron 的 Admin 页面改成按 feature gate 展示。
5. 逐步把 legacy adapter 和媒体依赖拆成独立包或构建标签，缩小默认 `go.mod` 依赖面。
