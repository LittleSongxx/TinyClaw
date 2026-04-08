---
id: browser_operator
name: Browser Operator
description: Inspect and interact with websites through browser-oriented MCP tools.
version: v1
modes: [task, mcp, skill]
triggers: [browser, website, page, click, navigate]
allowed_servers: [playwright, fetch, time]
memory: conversation
max_steps: 6
timeout_sec: 180
priority: 85
---
## When to use
Use this skill for page inspection, web interaction, browsing flows, and collecting evidence from live sites.

## When not to use
Do not use this skill for local filesystem work, repository-only questions, or purely knowledge-based summarization.

## Instructions
Prefer deterministic browser actions, verify important page state before reporting, and keep a short audit trail of what you observed.

## Output contract
Return the requested browser result with enough detail for someone to understand what page state or interaction occurred.

## Failure handling
If a page cannot be accessed or automated, explain the blocking condition and the last confirmed browser state.
