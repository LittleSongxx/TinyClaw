# TinyClaw Discord Guide

TinyClaw supports Discord through the Discord bot adapter.

This document keeps only the setup information that still fits the current TinyClaw repository layout.

## Required Variables

Configure these in `deploy/docker/.env`:

```env
DISCORD_BOT_TOKEN=your_discord_bot_token
TYPE=aliyun
DEFAULT_MODEL=qwen-max
ALIYUN_TOKEN=your_qwen_api_key
```

## Start TinyClaw

```bash
./scripts/start.sh
```

Then verify runtime status:

```bash
./scripts/status.sh
```

## How To Use

- direct message the bot for private chat
- mention the bot in a guild channel for group interaction

Common commands:

- `/help`
- `/clear`
- `/retry`
- `/mode`
- `/state`
- `/photo`
- `/video`

## Common Checks

If Discord messages are not answered, check:

- `DISCORD_BOT_TOKEN`
- whether the bot is invited to the target server
- whether the bot has permission to read and send messages
- container health and runtime logs
