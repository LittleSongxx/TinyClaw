# TinyClaw MCP / Function Call 说明

TinyClaw 可以通过 MCP 配置把外部工具接入到机器人能力中，并在运行时以 function calling 的方式使用这些工具。

这份文档只保留当前 TinyClaw 项目里仍然有用的 MCP 配置说明。

## 核心思路

你需要准备一份 MCP 配置 JSON，然后让 TinyClaw 在启动时加载它。

默认情况下，项目会读取：

```text
conf/mcp/mcp.json
```

如果你想使用自定义路径，可以设置：

```text
MCP_CONF_PATH
```

## 一个最小示例

```json
{
  "mcpServers": {
    "playwright": {
      "url": "http://localhost:8931/mcp",
      "description": "Browser automation and page interaction."
    }
  }
}
```

你也可以加入其他 MCP 服务，例如 GitHub、地图、检索等。

## 在当前项目里怎么启用

### 方式 1：直接使用默认配置文件

把配置写到：

```text
conf/mcp/mcp.json
```

然后在 `deploy/docker/.env` 中启用：

```env
USE_TOOLS=true
```

最后启动：

```bash
./scripts/start.sh
```

### 方式 2：使用自定义配置文件路径

如果你不想覆盖默认文件，可以在环境变量里指定：

```bash
export MCP_CONF_PATH=/path/to/your/mcp_config.json
```

然后再启动 TinyClaw。

## Docker 场景

如果你通过当前仓库的 Docker Compose 运行，推荐做法是：

- 把 MCP 配置文件放进仓库或映射卷可访问的位置
- 在 `deploy/docker/.env` 中设置 `USE_TOOLS=true`
- 如有需要，再补充 `MCP_CONF_PATH`

## 常见问题

### 日志里出现 MCP 连接失败

如果你看到类似：

```text
CheckSSEOrHTTP fail
```

通常表示：

- MCP 服务本身没启动
- `url` 地址不对
- TinyClaw 容器访问不到该地址

### 配置了 MCP 但机器人没调用工具

优先检查：

- `USE_TOOLS=true`
- 配置文件 JSON 是否合法
- MCP 服务是否真的可访问
- 当前提问是否足以触发工具调用
