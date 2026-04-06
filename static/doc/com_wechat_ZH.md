# TinyClaw 企业微信接入说明

TinyClaw 支持通过企业微信适配器接入企业微信。

这个适配器通过 TinyClaw 的 HTTP 服务接收企业微信回调。

## 需要的配置

在 `deploy/docker/.env` 中配置：

```env
COM_WECHAT_TOKEN=your_wecom_token
COM_WECHAT_ENCODING_AES_KEY=your_wecom_encoding_aes_key
COM_WECHAT_CORP_ID=your_wecom_corp_id
COM_WECHAT_SECRET=your_wecom_secret
COM_WECHAT_AGENT_ID=your_wecom_agent_id
TYPE=aliyun
DEFAULT_MODEL=qwen-max
ALIYUN_TOKEN=your_qwen_api_key
```

## 启动方式

```bash
./scripts/start.sh
```

## 回调路径

TinyClaw 中企业微信的回调路径是：

```text
/com/wechat
```

所以企业微信平台里的回调地址应指向：

```text
https://your-domain.example/com/wechat
```

## 如何使用

- 私聊应用
- 在支持的企业会话场景中使用

常用命令：

- `/help`
- `/clear`
- `/mode`
- `/state`
- `/photo`
- `/video`

## 常见检查项

如果企业微信没有正常回复，优先检查：

- 回调地址
- Token / AES Key / Corp ID / Agent 凭据
- 应用是否对目标成员可见
- 容器健康状态和运行日志
