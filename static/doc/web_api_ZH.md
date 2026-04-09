# TinyClaw Web API 说明

这份文档描述的是 `http/http.go` 当前真正暴露出来的 bot 侧 HTTP 接口。

基础地址就是 Bot HTTP 地址，例如：

```text
http://127.0.0.1:36060
```

## 响应格式

成功响应统一是：

```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

失败时 `code` 会变成非 0，`msg` 会带错误信息。

## 核心运行接口

| 方法 | 接口 | 作用 |
|---|---|---|
| `GET` | `/pong` | 健康检查 |
| `GET` | `/metrics` | Prometheus 指标 |
| `GET` | `/dashboard` | 运行统计和启动时间 |
| `GET` | `/command/get` | 当前生效的命令行差异参数 |
| `GET` | `/conf/get` | 当前运行配置快照 |
| `POST` | `/conf/update` | 更新一个配置字段 |
| `POST` | `/restart` | 带参数重启进程 |
| `POST` | `/stop` | 停止当前进程 |
| `GET` | `/log` | 持续输出 `log/tiny_claw.log` |

## 实时对话接口

### `POST /communicate`

这是当前最核心的 SSE 接口，负责普通聊天、图片/视频流程、`/task` 和 `/mcp`。

Query 参数：

| 参数 | 必填 | 说明 |
|---|---|---|
| `prompt` | 是 | 普通文本或斜杠命令 |
| `user_id` | 是 | 运行时用户 ID |

请求体：

- 可选的二进制负载，用于图片、音频等命令输入

响应：

- `text/event-stream`

常见命令：

- `/help`
- `/clear`
- `/retry`
- `/mode`
- `/state`
- `/photo`
- `/video`
- `/task`
- `/mcp`

## 用户与聊天记录接口

| 方法 | 接口 | 作用 |
|---|---|---|
| `POST` | `/user/token/add` | 给用户追加可用 token |
| `GET` | `/user/list` | 分页获取用户列表 |
| `DELETE` | `/user/delete?user_id=...` | 删除一个用户 |
| `POST` | `/user/insert/record` | 批量写入用户记录 |
| `GET` | `/record/list` | 分页获取聊天记录 |
| `DELETE` | `/record/delete?record_id=...` | 删除一条记录 |

### `POST /user/token/add`

请求体：

```json
{
  "user_id": "user123",
  "token": 100
}
```

### `GET /user/list`

Query 参数：

| 参数 | 必填 | 说明 |
|---|---|---|
| `page` | 否 | 服务端 / DB 默认值 |
| `page_size` | 否 | 服务端 / DB 默认值 |
| `user_id` | 否 | 按用户筛选 |

### `GET /record/list`

Query 参数：

| 参数 | 必填 | 说明 |
|---|---|---|
| `page` | 否 | 默认 `1` |
| `page_size` | 否 | 默认 `10` |
| `is_deleted` | 否 | `0`、`1` 或不传 |
| `user_id` | 否 | 按用户筛选 |
| `record_type` | 否 | 按记录类型筛选 |

## Agent 运行轨迹接口

这些接口支撑后台 `#/runs` 页面。

| 方法 | 接口 | 作用 |
|---|---|---|
| `GET` | `/run/list` | 分页获取运行列表 |
| `GET` | `/run/get?id=...` | 获取单条运行及步骤详情 |
| `POST` | `/run/replay` | 重放历史运行 |
| `DELETE` | `/run/delete` | 删除运行及其步骤 |

### `GET /run/list`

Query 参数：

| 参数 | 必填 | 说明 |
|---|---|---|
| `page` | 否 | 默认 `1` |
| `page_size` | 否 | 默认 `10`，也兼容 `pageSize` |
| `mode` | 否 | `task`、`mcp`、`skill` |
| `status` | 否 | `running`、`succeeded`、`failed` |
| `user_id` | 否 | 按用户筛选，也兼容 `userId` |

### `POST /run/replay`

接受的表单 / Query 参数：

| 参数 | 必填 | 说明 |
|---|---|---|
| `id` | 是 | 要重放的运行 ID |

### `DELETE /run/delete`

接受的表单 / Query 参数：

| 参数 | 必填 | 说明 |
|---|---|---|
| `id` | 是 | 推荐使用的运行 ID 字段 |
| `run_id` | 否 | 为兼容 Admin 代理也接受 |

## Skills 接口

这些接口支撑后台 `#/skills` 页面。

| 方法 | 接口 | 作用 |
|---|---|---|
| `GET` | `/skills/list` | 获取当前技能目录 |
| `GET` | `/skills/detail?id=...` | 获取单个技能详情 |
| `POST` | `/skills/reload` | 重新加载技能目录 |
| `GET` | `/skills/validate` | 校验当前技能目录并汇总 warning |

补充说明：

- 技能来源会区分 `local`、`builtin`、`legacy`
- 本地技能来自 `skills/*/SKILL.md`
- 校验结果会返回来源数量统计和 warning 列表

## MCP 接口

| 方法 | 接口 | 作用 |
|---|---|---|
| `GET` | `/mcp/get` | 读取当前 MCP 配置文件 |
| `GET` / `POST` | `/mcp/inspect` | 检查 MCP 配置可用性和安装状态 |
| `POST` | `/mcp/update?name=...` | 新增或更新一个 MCP 服务 |
| `DELETE` | `/mcp/delete?name=...` | 删除一个 MCP 服务 |
| `POST` | `/mcp/disable?name=...&disable=0|1` | 启用或禁用一个 MCP 服务 |
| `POST` | `/mcp/sync` | 清空当前客户端并重新初始化 MCP 注册 |

### `POST /mcp/update?name=...`

请求体是一个 `mcpParam.MCPConfig` 对象，例如：

```json
{
  "url": "http://playwright-mcp:8931/mcp",
  "description": "Browser automation and inspection."
}
```

### `GET|POST /mcp/inspect`

- `GET`：检查当前已经保存的配置
- `POST`：检查你提交的配置，但不落盘保存

返回结果会包含：

- 原始 `mcpServers`
- 每个服务的 availability 状态
- setup / runtime / secret 相关 warning

## Runtime / Knowledge 接口

当前统一 runtime 与 knowledge 相关 HTTP 接口包括：

| 方法 | 接口 | 作用 |
|---|---|---|
| `POST` | `/runs` | 统一发起 chat / task / skill / workflow run |
| `GET` | `/runs/{id}` | 获取单个 run 结果 |
| `GET` | `/tools/effective` | 查看当前运行时实际可用工具 |
| `GET` | `/skills/status` | 查看技能状态 |
| `GET` | `/memory/status` | 查看 memory 状态 |
| `GET` | `/knowledge/status` | 查看 knowledge 状态 |
| `POST` | `/knowledge/search` | 执行统一 knowledge 检索 |
| `POST` | `/knowledge/ingest` | 向统一 knowledge 库写入文本或文件 |

knowledge 管理接口包括：

| 方法 | 接口 | 作用 |
|---|---|---|
| `GET` | `/knowledge/files/list` | 统一知识库文件列表 |
| `POST` | `/knowledge/files/create` | 创建统一知识库文件 |
| `GET` | `/knowledge/files/get` | 读取统一知识库文件 |
| `DELETE` | `/knowledge/files/delete` | 删除统一知识库文件 |
| `POST` | `/knowledge/clear` | 清空统一知识库数据 |
| `GET` | `/knowledge/collections/list` | 列出 collection |
| `POST` | `/knowledge/collections/create` | 创建 collection |
| `GET` | `/knowledge/documents/list` | 列出 document |
| `GET` | `/knowledge/documents/get` | 获取单个 document |
| `POST` | `/knowledge/documents/create` | 创建文本或二进制 document |
| `DELETE` | `/knowledge/documents/delete` | 删除 document |
| `GET` | `/knowledge/jobs/list` | 获取 ingestion job 列表 |
| `POST` | `/knowledge/retrieval/debug` | 执行 retrieval debug |
| `GET` | `/knowledge/retrieval/runs/list` | 获取 retrieval run 列表 |
| `GET` | `/knowledge/retrieval/runs/get` | 获取单个 retrieval run |

当前自检脚本 `scripts/verify.sh` 重点依赖这些 knowledge 接口：

- `/knowledge/collections/list`
- `/knowledge/documents/create`
- `/knowledge/jobs/list`
- `/knowledge/retrieval/debug`

## Cron 接口

| 方法 | 接口 | 作用 |
|---|---|---|
| `GET` | `/cron/list` | 分页获取定时任务 |
| `POST` | `/cron/create` | 创建定时任务 |
| `POST` | `/cron/update` | 更新定时任务 |
| `POST` | `/cron/update_status` | 启用或禁用定时任务 |
| `DELETE` | `/cron/delete` | 删除定时任务 |

## 平台与其他接口

运行时还暴露了这些入口：

| 方法 | 接口 | 作用 |
|---|---|---|
| `GET` | `/image` | 图片读取辅助接口 |
| `POST` | `/com/wechat` | WeChat 通信入口 |
| `POST` | `/wechat` | WeChat Bot 入口 |
| `POST` | `/qq` | QQ Bot 入口 |
| `POST` | `/onebot` | OneBot 入口 |

## 实际使用说明

- Admin 使用的是自己的 `/bot/...` 代理路由，但最终转发到这里记录的 bot 侧接口
- 当前 `scripts/verify.sh` 会直接检查 `/pong`、`/metrics`、`/run/list` 和多组 Knowledge 接口
- 当前后台 `#/runs` 页面依赖 `/run/list`、`/run/get`、`/run/replay`、`/run/delete`
- 当前后台 `#/skills` 页面依赖 `/skills/list`、`/skills/detail`、`/skills/reload`、`/skills/validate`
