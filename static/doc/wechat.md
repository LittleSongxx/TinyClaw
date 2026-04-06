# TinyClaw WeChat Official Account Guide

TinyClaw supports WeChat Official Account integration through the WeChat adapter.

This adapter uses the TinyClaw HTTP service to receive callbacks.

## Required Variables

Configure these in `deploy/docker/.env`:

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

## Start TinyClaw

```bash
./scripts/start.sh
```

## Callback Path

The WeChat callback path in TinyClaw is:

```text
/wechat
```

So your WeChat platform callback URL should point to:

```text
https://your-domain.example/wechat
```

## Notes About `WECHAT_ACTIVE`

- `true`: proactive messaging mode when your WeChat setup allows it
- `false`: passive reply mode

## How To Use

- chat through the official account
- use bot commands inside supported message flows

Common commands:

- `/help`
- `/clear`
- `/mode`
- `/state`
- `/photo`
- `/video`

## Common Checks

If WeChat does not reply, check:

- callback URL
- app ID / app secret / token / AES key
- whether your official account configuration matches the TinyClaw callback path
- runtime logs and container health
