# TinyClaw Feishu / Lark Guide

Feishu / Lark long connection mode is the currently recommended TinyClaw integration path.

This document keeps only the setup that is actually relevant for the current TinyClaw project and no longer follows the older “many platforms + one old model” wording.

## Recommended Stack

- Platform: Feishu / Lark
- Model: `qwen-max` through Aliyun Bailian
- Deployment: Docker Compose
- Connection mode: Lark websocket long connection

This means:

- no public webhook is required
- no `cloudflared` tunnel is required
- no callback URL needs to be exposed

## Required Configuration

Edit:

`deploy/docker/.env`

At minimum, configure:

```env
BOT_NAME=TinyClawLark
LANG=zh
TYPE=aliyun
MEDIA_TYPE=aliyun
DEFAULT_MODEL=qwen-max
DB_TYPE=sqlite3

LARK_APP_ID=your_lark_app_id
LARK_APP_SECRET=your_lark_app_secret
ALIYUN_TOKEN=your_qwen_api_key
```

## Start TinyClaw

```bash
./scripts/start.sh
```

Then check status:

```bash
./scripts/status.sh
```

## What To Configure in Feishu Open Platform

You need to complete these steps in the Feishu developer console:

1. create the application
2. enable bot capability
3. set the correct visibility scope
4. grant message-related permissions
5. install the app into the target account or tenant scope

Because TinyClaw currently uses long connection mode, you do not need to configure a classic webhook callback URL.

## How To Verify It Works

Once startup succeeds, logs usually contain:

- `LarkBot Info`

That indicates TinyClaw has successfully connected to Feishu.

Then test with:

- direct private chat
- group chat with `@bot`

## Common Commands

The most commonly used commands in Feishu are:

- `/help`
- `/clear`
- `/retry`
- `/mode`
- `/state`
- `/photo`
- `/video`
- `/mcp`

## Common Issues

### The bot cannot be found in Feishu

Check:

- whether the app is installed
- whether the visibility scope includes your account
- whether bot capability is enabled

### The connection is up but messages get no reply

Check:

- `LARK_APP_ID` / `LARK_APP_SECRET`
- `ALIYUN_TOKEN`
- whether the bot was mentioned in a group
- container health

### I want to switch to another platform

No code change is usually needed. Update the platform credentials in `deploy/docker/.env` and restart the stack.
