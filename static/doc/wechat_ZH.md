# TinyClaw 微信公众号接入说明

TinyClaw 支持通过微信公众号适配器接入微信公众号。

这个适配器通过 TinyClaw 的 HTTP 服务接收微信回调。

## 需要的配置

在 `deploy/docker/.env` 中配置：

```env
WECHAT_APP_ID=your_wechat_app_id
WECHAT_APP_SECRET=your_wechat_app_secret
WECHAT_TOKEN=your_wechat_token
WECHAT_ENCODING_AES_KEY=your_wechat_encoding_aes_key
WECHAT_ACTIVE=false
TYPE=aliyun
DEFAULT_MODEL=qwen-max
ALIYUN_TOKEN=your_qwen_api_key
```

## 启动方式

```bash
./scripts/start.sh
```

## 回调路径

TinyClaw 中微信公众号的回调路径是：

```text
/wechat
```

所以微信平台里的回调地址应指向：

```text
https://your-domain.example/wechat
```

## `WECHAT_ACTIVE` 说明

- `true`：在你的微信配置允许时使用主动消息模式
- `false`：使用被动回复模式

## 如何使用

- 通过公众号与机器人对话
- 在支持的消息场景中使用命令

常用命令：

- `/help`
- `/clear`
- `/mode`
- `/state`
- `/photo`
- `/video`

## 常见检查项

如果微信公众号没有正常回复，优先检查：

- 回调地址
- App ID / App Secret / Token / EncodingAESKey
- 微信后台配置是否和 TinyClaw 回调路径一致
- 容器健康状态和运行日志
