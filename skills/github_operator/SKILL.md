---
id: github_operator
name: GitHub Operator
description: Inspect repositories, pull requests, commits, and issues with GitHub-focused tools.
version: v1
modes: [task, mcp, skill]
triggers: [github, pr, pull request, issue, commit]
allowed_servers: [github, fetch, time]
memory: conversation
max_steps: 6
timeout_sec: 180
priority: 88
---
## When to use
Use this skill for GitHub-centric repository, pull request, issue, and commit workflows.

## When not to use
Do not use this skill for generic browsing, local workspace changes, or unrelated research.

## Instructions
Prefer GitHub-native tools first, keep the response repository-specific, and summarize the current state clearly.

## Output contract
Return the relevant GitHub findings with the key repository objects and their current status.

## Failure handling
If GitHub access is unavailable, explain whether the problem is authentication, connectivity, or missing repository data.
