# TinyClaw QQ Guide

TinyClaw supports the official QQ bot adapter.

This adapter uses the TinyClaw HTTP service to receive QQ events.

## Required Variables

Set these in `deploy/docker/.env`:

```env
QQ_APP_ID=your_qq_app_id
QQ_APP_SECRET=your_qq_app_secret
TYPE=aliyun
DEFAULT_MODEL=qwen-max
ALIYUN_TOKEN=your_qwen_api_key
```

## Start TinyClaw

```bash
./scripts/start.sh
```

## Callback Path

The QQ callback path in TinyClaw is:

```text
/qq
```

So your QQ platform callback URL should point to:

```text
https://your-domain.example/qq
```

## How To Use

- private chat the bot directly
- mention the bot in group chat

Common commands:

- `/help`
- `/clear`
- `/mode`
- `/state`
- `/photo`
- `/video`

## Common Checks

If QQ delivery fails, check:

- `QQ_APP_ID` / `QQ_APP_SECRET`
- callback URL correctness
- whether the TinyClaw HTTP service is reachable
- runtime logs and container health
