#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

SCRIPT_HINT_CMD="./scripts/stop.sh --down"
SCRIPT_RUN_ID="${SCRIPT_RUN_ID:-$(date +%Y%m%d-%H%M%S)}"

cd "${REPO_ROOT}"
source_env_file "${ENV_FILE}"
ensure_script_log_dir

if [[ "${1:-}" == "--down" ]]; then
  down_log="${SCRIPT_LOG_DIR}/${SCRIPT_RUN_ID}-compose-down.log"
  run_with_log "Stopping the Compose stack" "${down_log}" \
    docker_compose down

  script_section "Done"
  script_ok "All TinyClaw containers have been stopped"
  script_kv "Log" "${down_log}"
  exit 0
fi

script_section "No Action Taken"
script_warn "stop.sh keeps the stack running unless you pass --down."
script_kv "Stop command" "./scripts/stop.sh --down"
script_kv "Status command" "./scripts/status.sh"

echo
compose_status_table
