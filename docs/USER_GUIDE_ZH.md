# TinyClaw 项目使用指南

这份文档面向你当前已经部署好的这套 TinyClaw 环境，重点不是讲“怎么从零开发”，而是讲“现在这套系统怎么稳定地用起来、管起来、排查起来”。

## 1. 当前这套环境是什么

你当前使用的是一套已经落地好的 TinyClaw 运行环境，核心特点如下：

- 机器人平台：飞书
- 大模型提供方：阿里云百炼
- 当前默认文本模型：`qwen-max`
- 当前默认多媒体类型：`aliyun`
- 部署方式：Docker Compose
- 运行目录：`/path/to/TinyClaw`
- 默认本地 Bot API 起始端口：`36060`
- 默认本地管理后台起始端口：`18080`
- 当前数据库：SQLite

这套飞书接入走的是长连接 `ws` 模式，不需要公网 webhook，也不需要 `cloudflared` 隧道。
在 Docker Desktop 里，容器会按 Compose 项目名 `tinyclaw` 分组显示，主服务会显示为 `app`。

## 2. 项目能做什么

TinyClaw 不是一个单一聊天机器人，而是一个“多平台接入 + 多模型调度 + 多模态能力 + 管理后台”的 AI 机器人系统。

你当前能直接用到的核心能力包括：

- 飞书私聊机器人，触发 Qwen 文本对话
- 飞书群聊里 `@机器人` 触发对话
- 文本聊天与上下文记忆
- 图片生成
- 视频生成
- 图片识别
- 语音相关能力
- Web API 调用
- 管理后台查看 Bot、用户、聊天记录、配置、日志
- MCP / Function Call / RAG / Cron 等扩展能力

## 3. 目录说明

你最常接触的是这个运行目录：

- [TinyClaw](/path/to/TinyClaw)

里面的重要文件如下：

- [`.env`](/path/to/TinyClaw/deploy/docker/.env)
  作用：运行配置文件，里面放平台凭据、模型类型、端口起始值等
- [docker-compose.yml](/path/to/TinyClaw/deploy/docker/docker-compose.yml)
  作用：容器编排文件
- [start.sh](/path/to/TinyClaw/scripts/start.sh)
  作用：启动服务，自动选择可用端口
- [stop.sh](/path/to/TinyClaw/scripts/stop.sh)
  作用：停止服务
- [status.sh](/path/to/TinyClaw/scripts/status.sh)
  作用：查看容器状态和当前端口
- [bootstrap-host.sh](/path/to/TinyClaw/scripts/bootstrap-host.sh)
  作用：补齐宿主机 Go / Docker 基础环境
- [data](/path/to/TinyClaw/data)
  作用：保存数据库和运行时配置快照
- [log](/path/to/TinyClaw/log)
  作用：保存运行日志

源码目录里你最关心的几个位置：

- [main.go](/path/to/TinyClaw/cmd/tinyclaw/main.go)
  主服务启动入口
- [admin/main.go](/path/to/TinyClaw/admin/main.go)
  管理后台启动入口
- [robot/robot.go](/path/to/TinyClaw/robot/robot.go)
  多平台机器人统一执行核心
- [robot/lark.go](/path/to/TinyClaw/robot/lark.go)
  飞书接入实现
- [http/http.go](/path/to/TinyClaw/http/http.go)
  Web API 入口
- [README_ZH.md](/path/to/TinyClaw/README_ZH.md)
  官方中文说明

## 4. 日常启动、停止、查看状态

所有命令都建议在运行目录下执行：

```bash
cd /path/to/TinyClaw
```

### 启动

```bash
./scripts/start.sh
```

作用：

- 构建镜像
- 自动选择空闲端口
- 启动 TinyClaw 和管理后台

### 停止

```bash
./scripts/stop.sh
```

### 查看状态

```bash
./scripts/status.sh
```

你会看到：

- 容器是否健康
- 当前映射端口
- 当前数据目录和日志目录

## 5. 当前访问入口

### 管理后台

地址：

```text
http://127.0.0.1:18080
```

当前默认账号：

```text
admin / admin
```

说明：

- 这是项目默认初始化账号
- 建议尽快修改密码

### 本地健康检查

```text
http://127.0.0.1:36060/pong
```

如果服务正常，会返回 `pong`。

### Metrics

```text
http://127.0.0.1:36060/metrics
```

可用于 Prometheus 抓取。

## 6. 飞书里怎么使用

### 私聊使用

直接在飞书里找到机器人并私聊即可。

### 群聊使用

在群里必须先 `@机器人`，否则机器人默认不会响应群消息。

### 文本对话

直接发送自然语言即可，例如：

```text
帮我总结一下这段话
```

### 常用命令

你当前常用的命令建议先记这些：

- `/help`
  查看帮助
- `/clear`
  清空当前上下文
- `/retry`
  重试上一次提问
- `/mode`
  查看当前模型配置
- `/state`
  查看当前会话/用户状态
- `/photo`
  生成图片
- `/edit_photo`
  编辑图片
- `/video`
  生成视频
- `/task`
  多代理协作任务
- `/mcp`
  调用多代理控制面板能力

### 模型切换命令

如果你后续想在聊天中切换模型，可用这些命令：

- `/txt_type`
- `/photo_type`
- `/video_type`
- `/rec_type`
- `/txt_model`
- `/img_model`
- `/video_model`
- `/rec_model`

你当前默认文本模型已经固定是 `qwen-max`，所以不改命令也会直接用 Qwen。

补充说明：

- 项目文档里有时会写 `/img_model`
- 有时也会写 `/photo_model`

如果你遇到文档和实际提示不一致，优先以机器人回复 `/help` 后显示的实际命令为准。

## 7. 管理后台怎么用

管理后台入口：

- [http://127.0.0.1:18080](http://127.0.0.1:18080)

后台里你最常用的页面通常是：

- Dashboard
  看整体运行情况
- Bots
  看 Bot 列表和配置
- BotUsers
  看用户信息
- BotChats
  看聊天记录
- Chat
  直接在后台和 Bot 对话
- Log
  查看日志
- MCP
  管理 MCP
- Cron
  管理定时任务
- RAG
  管理知识库文件

如果你只是日常使用，这几个入口最重要：

- `Chat`：后台里直接调试 Bot
- `BotChats`：看用户聊天记录
- `Log`：排错
- `RAG`：上传知识库文件

## 8. 当前数据存在哪里

所有持久化数据都在这里：

- [data](/path/to/TinyClaw/data)

当前你会看到几个关键文件：

- `tiny_claw.db`
  主机器人数据库
- `tiny_claw_admin.db`
  管理后台数据库
- `TinyClawLark*.json`
  当前运行配置快照

日志目录：

- [log](/path/to/TinyClaw/log)

当前主日志文件通常是：

- `tiny_claw.log`

## 9. 备份与迁移

如果你想备份这套环境，最核心的是备份下面两个目录：

- [data](/path/to/TinyClaw/data)
- [log](/path/to/TinyClaw/log)

最简单的备份方式：

```bash
cd /path/to
tar -czf tinyclaw-backup-$(date +%F).tar.gz TinyClaw/data TinyClaw/log TinyClaw/deploy/docker/.env
```

说明：

- `data` 决定聊天记录、后台账号、Bot 配置是否保留
- `.env` 决定平台凭据和模型配置是否保留
- `log` 不是必须，但排障很有用

## 10. 日志与排障

### 看容器状态

```bash
cd /path/to/TinyClaw
./scripts/status.sh
```

### 看容器日志

```bash
docker compose \
  -f deploy/docker/docker-compose.yml \
  --env-file deploy/docker/.env \
  --env-file deploy/docker/.env.runtime \
  logs --tail 200 app
```

实时看日志：

```bash
docker compose \
  -f deploy/docker/docker-compose.yml \
  --env-file deploy/docker/.env \
  --env-file deploy/docker/.env.runtime \
  logs -f app
```

### 看本地日志文件

```bash
tail -n 200 /path/to/TinyClaw/log/tiny_claw.log
```

### 看最近聊天记录

```bash
sqlite3 /path/to/TinyClaw/data/tiny_claw.db \
"select id,user_id,mode,substr(question,1,60),substr(answer,1,60) from records order by id desc limit 20;"
```

### 常见问题

#### 1. 飞书里机器人不回复

先检查：

- 飞书应用是否已安装到可见范围
- 私聊是否已开启
- 群聊是否正确 `@机器人`
- 容器是否健康
- 日志里是否出现 `LarkBot Info`

#### 2. 服务能启动，但看到 MCP 报错

如果日志里看到类似：

```text
CheckSSEOrHTTP fail, err: Get "http://localhost:8931/mcp": connect refused
```

这通常表示你没有配置 MCP 服务。它不是当前飞书 + Qwen 基础聊天的阻塞问题，可以先忽略。

#### 3. 端口冲突

`scripts/start.sh` 已经做了自动顺延处理。

例如：

- `36060` 被占用，会自动尝试 `36061`
- `18080` 被占用，会自动尝试 `18081`

最终使用哪个端口，以 `./scripts/status.sh` 输出为准。

#### 4. 后台登不上

先确认：

- 访问的是当前端口
- 管理容器健康
- `tiny_claw_admin.db` 没被误删

如果是你改过 `SESSION_KEY`，旧浏览器登录态会失效，重新登录即可。

## 11. 如何切换平台

你当前已切到飞书。以后如果你想切回 QQ、Telegram 等，不需要改业务代码，通常只要改 [`.env`](/path/to/TinyClaw/deploy/docker/.env)。

### 切平台的基本原则

- 飞书：设置 `LARK_APP_ID`、`LARK_APP_SECRET`
- QQ：设置 `QQ_APP_ID`、`QQ_APP_SECRET`
- Telegram：设置 `TELEGRAM_BOT_TOKEN`
- 不用的平台，对应凭据留空

改完后重启：

```bash
cd /path/to/TinyClaw
./scripts/stop.sh
./scripts/start.sh
```

## 12. 如何切换模型

当前是：

- `TYPE=aliyun`
- `DEFAULT_MODEL=qwen-max`
- `MEDIA_TYPE=aliyun`

如果你后续想切别的文本模型，通常改 [`.env`](/path/to/TinyClaw/deploy/docker/.env) 里的这些字段：

- `TYPE`
- `DEFAULT_MODEL`
- 对应平台的 Token

改完重启即可。

## 13. Web API 怎么用

除了飞书之外，这个项目也提供本地 HTTP API。

最常用的是：

- `POST /communicate`

示例：

```bash
curl -X POST \
  "http://127.0.0.1:36060/communicate?prompt=你好&user_id=12345"
```

健康检查：

```bash
curl http://127.0.0.1:36060/pong
```

查看记录：

```bash
curl "http://127.0.0.1:36060/record/list?user_id=12345&page=1&page_size=10"
```

适合场景：

- 自己写前端
- 本地联调
- 自动化测试

## 14. 你现在最常用的操作清单

如果你只是“把它当成自己在用的 AI 机器人系统”，最常用的其实就这些：

### 启动服务

```bash
cd /path/to/TinyClaw
./scripts/start.sh
```

### 看状态

```bash
./scripts/status.sh
```

### 看日志

```bash
docker compose \
  -f deploy/docker/docker-compose.yml \
  --env-file deploy/docker/.env \
  --env-file deploy/docker/.env.runtime \
  logs -f app
```

### 在飞书里聊天

- 私聊：直接发送消息
- 群聊：先 `@机器人`

### 进后台

```text
http://127.0.0.1:18080
```

### 停服务

```bash
./scripts/stop.sh
```

## 15. 进阶建议

如果你接下来准备长期用这套环境，我建议优先做这几件事：

- 修改后台默认密码
- 备份 `data` 和 `.env`
- 明确哪些飞书群允许使用机器人
- 决定是否开启 RAG
- 决定是否接 MCP
- 决定是否保留图片/视频能力的额外模型配置

## 16. 推荐阅读

如果你要更深入使用项目，可以继续看这些专题文档：

- [README_ZH.md](/path/to/TinyClaw/README_ZH.md)
- [lark_ZH.md](/path/to/TinyClaw/static/doc/lark_ZH.md)
- [web_api_ZH.md](/path/to/TinyClaw/static/doc/web_api_ZH.md)

如果你希望，我下一步还可以继续帮你把这份指南再升级成两种版本中的任意一个：

- “极简版日常操作手册”
- “管理员版运维手册”
