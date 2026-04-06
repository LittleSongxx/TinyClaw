# TinyClaw Discord 接入说明

TinyClaw 支持通过 Discord 机器人适配器接入 Discord。

这份文档只保留当前仓库结构下仍然有用的接入信息。

## 需要的配置

在 `deploy/docker/.env` 中至少配置：

```env
DISCORD_BOT_TOKEN=your_discord_bot_token
TYPE=aliyun
DEFAULT_MODEL=qwen-max
ALIYUN_TOKEN=your_qwen_api_key
```

## 启动方式

```bash
./scripts/start.sh
```

然后查看状态：

```bash
./scripts/status.sh
```

## 如何使用

- 私聊机器人，进行一对一对话
- 在服务器频道里 `@机器人`，进行群聊交互

常用命令：

- `/help`
- `/clear`
- `/retry`
- `/mode`
- `/state`
- `/photo`
- `/video`

## 常见检查项

如果 Discord 不回复，优先检查：

- `DISCORD_BOT_TOKEN`
- 机器人是否已加入目标服务器
- 是否具有读写消息权限
- 容器是否健康、日志是否报错
