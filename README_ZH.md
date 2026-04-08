# TinyClaw

TinyClaw 是一个基于 Go 的多平台 AI 机器人项目，用来把聊天平台、Web API 和大模型能力接到一套统一的机器人内核上。

这个仓库现在已经按我自己的项目方式整理过，当前推荐的使用方式是：

- 平台：飞书
- 模型：阿里云百炼 `qwen-max`
- 部署：Docker Compose
- 应用数据：SQLite
- Agent / RAG v2：PostgreSQL `pgvector` + Redis + MinIO
- 管理方式：内置 Admin 后台

仓库里仍然保留了其他平台和模型适配器，但 README 不再把所有历史能力都堆在首页，优先围绕当前可直接落地的 TinyClaw 使用方式来写。

## 核心能力

- 多平台机器人接入，统一走同一套消息处理内核
- 文本对话、上下文记忆、基础命令体系
- 图片、音频、视频相关能力
- Web API 调用入口
- RAG、MCP / Function Call、Skills、Cron、指标监控
- 独立 Admin 后台，支持配置、运行轨迹、Skills、记录、用户和日志查看

## 当前推荐方案

如果你只是想把项目跑起来，推荐直接使用这套组合：

- 飞书机器人
- Qwen 文本模型
- Docker Compose
- SQLite + PostgreSQL `pgvector` + Redis + MinIO

这也是我当前仓库内已经完成验证的运行方式。

## 目录结构

```text
TinyClaw/
├─ cmd/tinyclaw/          主程序入口
├─ admin/                 管理后台服务与前端
├─ robot/                 平台适配层
├─ llm/                   模型调用层
├─ http/                  Bot HTTP API 与运行控制接口
├─ skill/                 技能目录加载、校验与组装逻辑
├─ skills/                当前仓库自带的本地 SKILL.md 技能定义
├─ conf/                  配置定义与默认配置
├─ deploy/docker/         Docker 部署文件
├─ scripts/               启动、停止、发布、构建脚本
├─ docs/                  面向当前部署的使用手册
├─ static/doc/            各平台/功能专题文档
├─ data/                  运行数据
└─ log/                   运行日志
```

## 快速开始

1. 克隆仓库

```bash
git clone https://github.com/LittleSongxx/TinyClaw.git
cd TinyClaw
```

2. 准备配置文件

```bash
cp deploy/docker/.env.example deploy/docker/.env
```

然后按你的实际平台和模型修改 `deploy/docker/.env`。

如果你要启用仓库默认带的地图、联网搜索或 GitHub MCP 服务，还需要补这几个密钥：

```env
AMAP_MAPS_API_KEY=
BOCHA_API_KEY=
GITHUB_PERSONAL_ACCESS_TOKEN=
```

飞书 + Qwen 的最小配置示例：

```env
BOT_NAME=TinyClawLark
LANG=zh
TYPE=aliyun
MEDIA_TYPE=aliyun
DEFAULT_MODEL=qwen-max
DB_TYPE=sqlite3

LARK_APP_ID=your_lark_app_id
LARK_APP_SECRET=your_lark_app_secret
ALIYUN_TOKEN=your_qwen_api_key
```

3. 启动服务

```bash
./scripts/start.sh
```

4. 查看状态

```bash
./scripts/status.sh
```

5. 执行自检

```bash
./scripts/verify.sh
```

如果你希望把 `/task`、`/mcp` 和 `/run/replay` 也一起做一轮真实在线验证：

```bash
./scripts/verify.sh --full
```

6. 安全停止辅助脚本

```bash
./scripts/stop.sh
```

这个命令现在默认不会停掉容器，只会提醒你当前启用了自启动，并输出栈状态。

如果你确实要主动停止整套 Docker Compose：

```bash
./scripts/stop.sh --down
```

`./scripts/start.sh` 现在会在当前栈已运行时复用已有端口，先拉起依赖服务，再单独重建 `app`。第一次启动或 Go 依赖变化后，`app` 的 build 阶段可能会花一些时间。

## 运行入口

- Bot HTTP：默认从 `36060` 起自动选择空闲端口
- Admin：默认从 `18080` 起自动选择空闲端口
- 健康检查：`/pong`
- 指标：`/metrics`
- Agent 运行轨迹页：Admin `#/runs`
- Skills 管理页：Admin `#/skills`
- RAG 工作台：Admin `#/rag`

实际使用端口以 `./scripts/status.sh` 输出为准。

## 默认 MCP 与 Skills 结构

默认情况下，TinyClaw 会从 `conf/mcp/mcp.json` 读取 MCP 配置。当前仓库已经预置这些服务：

- `playwright`
- `filesystem`
- `fetch`
- `time`
- `memory`
- `arxiv`
- `amap`
- `bocha-search`
- `github`

当前仓库自带的本地技能目录在 `skills/`，默认包含：

- `general_research`
- `browser_operator`
- `workspace_operator`
- `github_operator`

运行时还会基于已注册的 MCP 工具自动补出 builtin / legacy 兜底技能。后台里可以分别通过 `#/mcp` 和 `#/skills` 查看配置、检查可用性、校验技能目录并执行 reload。

## 容器自启动

[deploy/docker/docker-compose.yml](/home/song/code/Agent/TinyClaw/deploy/docker/docker-compose.yml) 里的服务已经统一设置了 `restart: unless-stopped`。

这表示：

- 只要你执行过一次 `./scripts/start.sh`，后续 Docker 守护进程或主机重启后，容器会自动恢复运行
- 其他机器只要沿用当前仓库的 Compose 部署文件，也会直接继承这套自启动策略

如果你部署在普通 Linux 主机上，还需要确保 Docker 服务本身是开机自启：

```bash
sudo systemctl enable --now docker
```

如果你使用的是 Docker Desktop，则以 Docker Desktop 自身的启动设置为准。

## 开发与构建

构建主程序：

```bash
go build ./cmd/tinyclaw
```

构建后台：

```bash
go build -o /tmp/TinyClawAdmin ./admin
```

使用 Makefile：

```bash
make build
make build-admin
make test
```

## 文档入口

- 当前部署使用手册：[docs/USER_GUIDE_ZH.md](docs/USER_GUIDE_ZH.md)
- 飞书接入说明：[static/doc/lark_ZH.md](static/doc/lark_ZH.md)
- Web API 说明：[static/doc/web_api_ZH.md](static/doc/web_api_ZH.md)
- Admin 说明：[static/doc/admin_ZH.md](static/doc/admin_ZH.md)
- MCP / Skills 说明：[static/doc/functioncall_ZH.md](static/doc/functioncall_ZH.md)
- RAG 说明：[static/doc/rag_ZH.md](static/doc/rag_ZH.md)
- 参数说明：[static/doc/param_conf_ZH.md](static/doc/param_conf_ZH.md)

## 说明

- 这个仓库已经从原来的上游 fork 体系迁到了我自己的仓库路径和依赖体系。
- 首页 README 现在只保留高频、稳定、对当前项目真正有用的内容。
- 各平台的专题文档还在 `static/doc/` 下保留，后续会继续按 TinyClaw 的定位逐步精简。
