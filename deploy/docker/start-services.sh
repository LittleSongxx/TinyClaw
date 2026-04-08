#!/usr/bin/env bash
set -euo pipefail

: "${HOME:=/app/data/home}"
: "${XDG_CACHE_HOME:=/app/data/.cache}"

mkdir -p "${HOME}" "${XDG_CACHE_HOME}" /app/data/mcp/memory /app/data/mcp/arxiv /app/data/sessions /app/log

/app/TinyClawAdmin &
admin_pid=$!

/app/start-tinyclaw.sh &
bot_pid=$!

cleanup() {
  kill "${admin_pid}" "${bot_pid}" 2>/dev/null || true
}

trap cleanup EXIT INT TERM

wait -n "${admin_pid}" "${bot_pid}"
status=$?

cleanup
wait || true

exit "${status}"
