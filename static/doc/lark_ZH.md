# TinyClaw 飞书接入说明

TinyClaw 当前最推荐的接入方式就是飞书长连接模式。

这份文档只保留 TinyClaw 当前项目下真正需要的飞书配置步骤，不再使用旧的“多平台 + 指定旧模型”的写法。

## 当前推荐组合

- 平台：飞书
- 模型：阿里云百炼 `qwen-max`
- 部署：Docker Compose
- 连接方式：飞书 `ws` 长连接

这意味着：

- 不需要公网 webhook
- 不需要 `cloudflared`
- 不需要把回调地址暴露到公网

## 需要准备的配置

编辑：

`deploy/docker/.env`

至少准备这些变量：

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

## 启动方式

```bash
./scripts/start.sh
```

然后查看状态：

```bash
./scripts/status.sh
```

## 飞书开放平台需要做什么

你需要在飞书开放平台完成这几件事：

1. 创建应用
2. 添加机器人能力
3. 打开应用可见范围
4. 给机器人授予消息相关权限
5. 把应用安装到你要测试的账号或组织范围

因为 TinyClaw 当前使用的是长连接模式，所以不需要再配置传统 webhook 回调地址。

## 如何验证接入成功

当服务启动成功后，日志里通常会出现：

- `LarkBot Info`

这表示 TinyClaw 已经成功连接飞书。

然后你可以这样测试：

- 私聊机器人，直接发送消息
- 群聊里先 `@机器人` 再发送消息

## 常用命令

你在飞书里最常用的通常是：

- `/help`
- `/clear`
- `/retry`
- `/mode`
- `/state`
- `/photo`
- `/video`
- `/mcp`

## 常见问题

### 飞书里搜不到机器人

优先检查：

- 应用是否已安装
- 可见范围是否包含当前账号
- 机器人能力是否启用

### 能连上，但消息不回复

优先检查：

- `LARK_APP_ID` / `LARK_APP_SECRET` 是否正确
- `ALIYUN_TOKEN` 是否可用
- 群聊里是否正确 `@机器人`
- 容器是否健康

### 想切别的平台

TinyClaw 代码层不需要改，通常只要修改 `deploy/docker/.env` 里的平台凭据并重启即可。
