---
id: general_research
name: General Research
description: Research a topic with evidence from web, academic, time, and memory tools before answering.
version: v1
modes: [task, mcp, skill]
triggers: [research, search, find, compare, summarize]
allowed_servers: [fetch, bocha-search, arxiv, time, memory]
memory: both
max_steps: 8
timeout_sec: 180
priority: 90
---
## When to use
Use this skill for research-heavy questions, comparisons, fact gathering, and evidence-backed summaries.

## When not to use
Do not use this skill for browser automation, local file operations, or GitHub-only workflows.

## Instructions
Gather only the evidence needed to answer the request, prefer direct sources when possible, and clearly separate findings from assumptions.

## Output contract
Return a concise answer with the strongest evidence first, then note uncertainty or missing data.

## Failure handling
If retrieval is incomplete, explain what could not be verified and provide the best grounded partial answer.
