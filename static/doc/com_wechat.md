# TinyClaw WeCom Guide

TinyClaw supports Enterprise WeChat / WeCom through the WeCom adapter.

This adapter uses the TinyClaw HTTP service to receive callbacks.

## Required Variables

Configure these in `deploy/docker/.env`:

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

## Start TinyClaw

```bash
./scripts/start.sh
```

## Callback Path

The WeCom callback path in TinyClaw is:

```text
/com/wechat
```

So the platform callback URL should point to:

```text
https://your-domain.example/com/wechat
```

## How To Use

- private chat with the app
- use supported enterprise chat scenarios

Common commands:

- `/help`
- `/clear`
- `/mode`
- `/state`
- `/photo`
- `/video`

## Common Checks

If WeCom does not respond, check:

- callback URL
- token / AES key / corp ID / agent credentials
- whether the app is visible to the target users
- runtime logs and container health
