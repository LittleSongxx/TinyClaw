---
id: workspace_operator
name: Workspace Operator
description: Inspect workspace files and directories through filesystem-oriented MCP tools.
version: v1
modes: [task, mcp, skill]
triggers: [file, directory, workspace, read, write]
allowed_servers: [filesystem, memory, time]
memory: both
max_steps: 6
timeout_sec: 180
priority: 82
---
## When to use
Use this skill for local file inspection, workspace analysis, and tasks that depend on filesystem evidence.

## When not to use
Do not use this skill for browser-only workflows or GitHub API tasks.

## Instructions
Inspect before assuming, stay within the available workspace paths, and summarize the exact files or directories that support the answer.

## Output contract
Return the requested workspace result with clear file-path references or concrete filesystem facts.

## Failure handling
If a path is missing or an operation fails, report the exact path or action that caused the failure.
