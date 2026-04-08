# TinyClaw MCP / Skills / Function Call 说明

TinyClaw 当前把工具能力分成两层：

- `MCP`：外部工具的接入和注册层
- `Skills`：在 MCP 工具之上再做一层任务级编排，负责限定工具集合、提示词结构、记忆策略和执行约束

你可以把它理解成：

- `conf/mcp/mcp.json` 决定“系统里有哪些工具”
- `skills/` 决定“这些工具如何按任务场景被组织成技能”
- `USE_TOOLS=true` 是当前部署里推荐保持开启的模型侧工具开关

## 当前仓库默认行为

默认情况下，TinyClaw 会从这里读取 MCP 配置：

```text
conf/mcp/mcp.json
```

如果你想换成自定义路径，可以设置：

```text
MCP_CONF_PATH
```

当前仓库默认的 MCP 配置文件里已经带了这些服务：

- `playwright`
- `filesystem`
- `fetch`
- `time`
- `memory`
- `arxiv`
- `amap`
- `bocha-search`
- `github`

当前仓库 `skills/` 目录里默认带了这些本地技能：

- `general_research`
- `browser_operator`
- `workspace_operator`
- `github_operator`

## Skills 是怎么生成的

运行时的技能目录来自三个来源：

1. `skills/*/SKILL.md` 里的本地技能
2. 代码里生成的 builtin fallback 技能
3. 基于当前已注册 MCP 工具自动生成的 legacy proxy 技能

需要注意的是：

- 本地 `SKILL.md` 是当前项目最推荐的技能维护入口
- builtin 技能只是在缺少同名本地技能时才会补出来
- legacy 技能是兼容层，确保每个已注册 MCP 服务至少都有一个可兜底的技能入口

此外，运行时还会生成一个总兜底技能：

- `legacy_all_tools_proxy`

## 在当前项目里怎么启用

建议在 `deploy/docker/.env` 中保持开启：

```env
USE_TOOLS=true
```

如果你要使用自定义 MCP 配置路径：

```bash
export MCP_CONF_PATH=/path/to/your/mcp_config.json
```

然后正常启动：

```bash
./scripts/start.sh
```

## 默认带的密钥型 MCP 服务

当前默认 `conf/mcp/mcp.json` 里已经带了几个依赖密钥的服务，因此 `.env` 里通常还要补这些值：

```env
AMAP_MAPS_API_KEY=
BOCHA_API_KEY=
GITHUB_PERSONAL_ACCESS_TOKEN=
```

如果这些值为空，常见现象是：

- MCP 配置里能看到对应服务
- 后台 `MCP / Skills` 页面会出现 warning
- 对应工具或技能在运行时不可用

## Docker 场景的推荐做法

如果你使用当前仓库自带的 Docker Compose，推荐保持这套方式：

- MCP 默认继续使用 `conf/mcp/mcp.json`
- 在 `deploy/docker/.env` 里保持 `USE_TOOLS=true`
- 如果启用了外部服务，补齐对应密钥
- Playwright 优先使用仓库自带容器地址 `http://playwright-mcp:8931/mcp`
- 本地技能统一维护在 `skills/`

## Admin 后台对应页面

当前后台把这两层能力分开管理：

- `#/mcp`
  查看当前 MCP 配置、预置模板和可用性检查结果
- `#/skills`
  查看技能目录、按来源筛选、执行校验、reload 技能目录，并查看技能详情

## 本地技能文件怎么写

本地技能文件格式是 `SKILL.md`，由 YAML frontmatter 和固定章节组成。

每个技能至少需要定义：

- `id`
- `name`
- `description`
- `modes`
- `memory`
- `When to use`
- `When not to use`
- `Instructions`
- `Output contract`
- `Failure handling`

运行时加载技能目录时会对这些字段做校验。

## 常见问题

### 日志里出现 MCP 连接失败

通常是以下几类原因：

- 容器里缺少对应 MCP 命令
- MCP 的 URL 地址不对
- `app` 容器访问不到对应服务
- 需要的密钥还没填

### Skills 页面出现校验 warning

优先检查：

- `skills/*/SKILL.md` 的 frontmatter 是否仍然合法
- 必需章节是否都还在
- 技能引用的 MCP server / tool 当前是否存在
- `conf/mcp/mcp.json` 是否可读且 JSON 合法

### 配了 MCP 但机器人没调用工具

优先确认：

- 当前维护中的 Docker 默认仍然保持 `USE_TOOLS=true`
- MCP 服务确实已经注册成功
- 当前任务是否足以触发工具调用
- 对应技能最终是否还能解析到可用工具
