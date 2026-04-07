#!/usr/bin/env bash
set -euo pipefail

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
