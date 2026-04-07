# TinyClaw Admin 管理平台

`TinyClaw Admin` 是 TinyClaw 的内置管理后台，用来查看运行状态、管理机器人配置、查看聊天记录、维护用户、RAG、MCP 和定时任务。

这份文档只保留当前 TinyClaw 项目下真正有用的管理后台说明，不再沿用早期针对某个单一平台机器人的写法。

## 当前推荐方式

如果你已经按仓库当前结构部署，最推荐直接使用 Docker Compose 启动整套服务：

```bash
./scripts/start.sh
```

管理后台会和主服务一起启动。

默认管理后台内部端口是 `18080`，实际映射到宿主机的端口以：

```bash
./scripts/status.sh
```

输出为准。

## 后台登录

首次初始化时，默认账号是：

- 用户名：`admin`
- 密码：`admin`

建议首次登录后立即修改密码。

## 后台能做什么

你通常会在后台里使用这些模块：

- `Dashboard`
  查看整体运行状态、消息量、用户量和基础统计
- `Bots`
  查看和修改 Bot 配置
- `BotUsers`
  查看用户列表、用户模式和额度
- `BotChats`
  查看聊天记录
- `Chat`
  直接在后台和当前 Bot 调试对话
- `Log`
  查看运行日志
- `RAG`
  上传和管理知识库文件
- `MCP`
  查看和管理 MCP 服务配置
- `Cron`
  查看和管理定时任务

## 当前项目里的运行方式

### 方式 1：通过 Docker Compose

这是当前仓库默认方式。

```bash
./scripts/start.sh
./scripts/status.sh
./scripts/stop.sh
```

`./scripts/stop.sh` 现在是安全辅助脚本，默认不会停掉容器。
只有在你明确要停整套 Compose 时，才使用 `./scripts/stop.sh --down`。

### 方式 2：单独运行 Admin

如果你只想单独调试后台，也可以手动构建并启动：

```bash
go build -o /tmp/TinyClawAdmin ./admin
```

然后自行提供以下环境变量：

- `DB_TYPE`
- `DB_CONF`
- `SESSION_KEY`
- `ADMIN_PORT`

示例：

```bash
DB_TYPE=sqlite3 \
DB_CONF=./data/tiny_claw_admin.db \
SESSION_KEY=replace-with-your-session-key \
ADMIN_PORT=18080 \
/tmp/TinyClawAdmin
```

## 关键配置项

| 变量名 | 说明 | 示例 |
|---|---|---|
| `DB_TYPE` | 后台数据库类型 | `sqlite3` |
| `DB_CONF` | 后台数据库文件或连接串 | `./data/tiny_claw_admin.db` |
| `SESSION_KEY` | 登录态签名密钥 | 随机长字符串 |
| `ADMIN_PORT` | 后台监听端口 | `18080` |

## 和主服务的关系

`TinyClaw Admin` 不是独立产品，它依赖 TinyClaw 主服务的运行数据和配置体系。

你通常需要一起关注：

- 主服务数据库：`data/tiny_claw.db`
- 后台数据库：`data/tiny_claw_admin.db`
- 主日志：`log/tiny_claw.log`
- 部署配置：`deploy/docker/.env`

## 常见问题

### 后台打不开

优先检查：

- 容器是否健康
- 当前映射端口是否变化
- `SESSION_KEY` 是否被改过
- `tiny_claw_admin.db` 是否还在

### 登录态突然失效

通常是：

- 你重建了环境
- 修改了 `SESSION_KEY`
- 浏览器里还是旧 cookie

这种情况重新登录即可。

### 后台能打开，但看不到机器人数据

优先确认：

- TinyClaw 主服务是否已经启动
- 主服务是否真的在写 `data/tiny_claw.db`
- 当前环境变量是否指向了正确的数据目录
