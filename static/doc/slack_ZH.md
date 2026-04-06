# TinyClaw Slack 接入说明

TinyClaw 支持通过 Slack 机器人适配器接入 Slack。

在当前仓库结构下，推荐直接通过 `deploy/docker/.env` 配置，然后使用标准 TinyClaw 启动方式运行。

## 需要的配置

```env
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_APP_TOKEN=xapp-your-app-token
TYPE=aliyun
DEFAULT_MODEL=qwen-max
ALIYUN_TOKEN=your_qwen_api_key
```

## 启动方式

```bash
./scripts/start.sh
```

## Slack 平台侧准备

在 Slack 平台里，至少确认：

- 应用已经安装到目标工作区
- 已获得 Bot Token 和 App Token
- 应用具备读写消息所需权限

## 如何使用

- 私聊机器人
- 频道中 `@机器人`

常用命令：

- `/help`
- `/clear`
- `/retry`
- `/mode`
- `/state`
- `/photo`
- `/video`

## 常见检查项

如果 Slack 不回复，优先检查：

- `SLACK_BOT_TOKEN`
- `SLACK_APP_TOKEN`
- 工作区安装状态和权限范围
- 容器健康状态与运行日志
