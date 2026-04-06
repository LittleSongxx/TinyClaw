# TinyClaw QQ 接入说明

TinyClaw 支持 QQ 官方机器人接入。

QQ 适配器通过 TinyClaw 的 HTTP 服务接收回调事件。

## 需要的配置

在 `deploy/docker/.env` 中至少配置：

```env
QQ_APP_ID=your_qq_app_id
QQ_APP_SECRET=your_qq_app_secret
TYPE=aliyun
DEFAULT_MODEL=qwen-max
ALIYUN_TOKEN=your_qwen_api_key
```

## 启动方式

```bash
./scripts/start.sh
```

## 回调路径

TinyClaw 中 QQ 的回调路径是：

```text
/qq
```

所以在 QQ 开放平台里配置回调地址时，应指向：

```text
https://your-domain.example/qq
```

## 如何使用

- 私聊机器人直接对话
- 群聊中 `@机器人`

常用命令：

- `/help`
- `/clear`
- `/mode`
- `/state`
- `/photo`
- `/video`

## 常见检查项

如果 QQ 消息没有正常收发，优先检查：

- `QQ_APP_ID` / `QQ_APP_SECRET`
- 回调地址是否正确
- TinyClaw HTTP 服务是否可达
- 容器健康状态和运行日志
