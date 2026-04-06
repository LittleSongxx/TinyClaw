# TinyClaw MCP / Function Calling Guide

TinyClaw can connect external tools through MCP configuration and expose them to the runtime as callable tools.

This document keeps only the MCP setup that is still relevant in the current TinyClaw repository.

## Core Idea

Prepare an MCP JSON config and let TinyClaw load it during startup.

By default, the project reads:

```text
conf/mcp/mcp.json
```

If you want a custom file path, set:

```text
MCP_CONF_PATH
```

## Minimal Example

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

You can add more MCP services such as GitHub, map, search, or other custom integrations.

## How To Enable It

### Option 1: Use the Default Config File

Write your MCP config to:

```text
conf/mcp/mcp.json
```

Then enable tools in `deploy/docker/.env`:

```env
USE_TOOLS=true
```

Finally start TinyClaw:

```bash
./scripts/start.sh
```

### Option 2: Use a Custom Config Path

If you do not want to overwrite the default file:

```bash
export MCP_CONF_PATH=/path/to/your/mcp_config.json
```

Then start TinyClaw normally.

## Docker Usage

With the current Docker Compose layout, the recommended approach is:

- place the MCP config where the runtime can access it
- set `USE_TOOLS=true` in `deploy/docker/.env`
- optionally set `MCP_CONF_PATH` if you use a non-default file

## Common Issues

### Logs show MCP connection failures

If you see something like:

```text
CheckSSEOrHTTP fail
```

It usually means:

- the MCP service itself is not running
- the configured `url` is wrong
- the TinyClaw container cannot reach that address

### MCP is configured but the bot does not call tools

Check:

- `USE_TOOLS=true`
- valid JSON config
- actual MCP service reachability
- whether the prompt is suitable for tool invocation
