# TinyClaw 钉钉接入说明

TinyClaw 支持通过钉钉机器人适配器接入钉钉。

当前仓库推荐的方式是：把钉钉凭据写进 `deploy/docker/.env`，再用标准 TinyClaw 启动流程运行。

## 需要的配置

```env
DING_CLIENT_ID=your_dingtalk_client_id
DING_CLIENT_SECRET=your_dingtalk_client_secret
TYPE=aliyun
DEFAULT_MODEL=qwen-max
ALIYUN_TOKEN=your_qwen_api_key
```

## 启动方式

```bash
./scripts/start.sh
```

## 钉钉平台侧准备

需要至少确认：

- 机器人能力已开启
- 消息相关权限已授予
- 钉钉侧连接方式与应用配置正确

## 如何使用

- 私聊机器人
- 在支持的群聊场景里使用机器人

常用命令：

- `/help`
- `/clear`
- `/mode`
- `/state`
- `/photo`
- `/video`

## 常见检查项

如果钉钉消息没有正常响应，优先检查：

- `DING_CLIENT_ID`
- `DING_CLIENT_SECRET`
- 钉钉应用权限
- 容器健康状态与运行日志
