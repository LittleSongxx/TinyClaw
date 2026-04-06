# TinyClaw Slack Guide

TinyClaw supports Slack through the Slack bot adapter.

For the current project layout, configure Slack through `deploy/docker/.env` and start the standard TinyClaw stack.

## Required Variables

```env
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_APP_TOKEN=xapp-your-app-token
TYPE=aliyun
DEFAULT_MODEL=qwen-max
ALIYUN_TOKEN=your_qwen_api_key
```

## Start TinyClaw

```bash
./scripts/start.sh
```

## Slack App Side

On the Slack side, make sure:

- the app is installed into the target workspace
- the bot token and app token are both issued
- the app has the required scopes to read and send messages

## How To Use

- direct message the bot
- mention the bot in a channel

Common commands:

- `/help`
- `/clear`
- `/retry`
- `/mode`
- `/state`
- `/photo`
- `/video`

## Common Checks

If Slack does not respond, check:

- `SLACK_BOT_TOKEN`
- `SLACK_APP_TOKEN`
- workspace installation and scopes
- runtime logs and container health
