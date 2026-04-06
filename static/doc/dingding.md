# TinyClaw DingTalk Guide

TinyClaw supports DingTalk through the DingTalk bot adapter.

The current project flow is to configure credentials in `deploy/docker/.env` and start the standard TinyClaw runtime.

## Required Variables

```env
DING_CLIENT_ID=your_dingtalk_client_id
DING_CLIENT_SECRET=your_dingtalk_client_secret
TYPE=aliyun
DEFAULT_MODEL=qwen-max
ALIYUN_TOKEN=your_qwen_api_key
```

## Start TinyClaw

```bash
./scripts/start.sh
```

## DingTalk Side

Make sure your DingTalk app is correctly configured with:

- bot capability enabled
- required message permissions
- the expected connection mode on the DingTalk side

## How To Use

- private chat the bot
- use it in supported group scenarios

Common commands:

- `/help`
- `/clear`
- `/mode`
- `/state`
- `/photo`
- `/video`

## Common Checks

If DingTalk does not respond, check:

- `DING_CLIENT_ID`
- `DING_CLIENT_SECRET`
- DingTalk-side app permissions
- runtime logs and container health
