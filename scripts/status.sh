#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

SCRIPT_HINT_CMD="./scripts/status.sh"

cd "${REPO_ROOT}"
source_env_file "${ENV_FILE}"

if [[ -f "${RUNTIME_FILE}" ]]; then
  source_env_file "${RUNTIME_FILE}"
fi

script_section "Compose Services"
if ! compose_status_table; then
  script_error "Failed to query Compose services."
  exit 1
fi

echo
script_section "Runtime"
if [[ -n "${HOST_HTTP_PORT:-}" ]]; then
  script_kv "HTTP" "http://127.0.0.1:${HOST_HTTP_PORT}"
fi
if [[ -n "${HOST_ADMIN_PORT:-}" ]]; then
  script_kv "Admin" "http://127.0.0.1:${HOST_ADMIN_PORT}"
fi
script_kv "Auto-start" "enabled (restart: unless-stopped)"
script_kv "Verify" "./scripts/verify.sh"

if [[ -f "${PUBLIC_URL_FILE}" ]]; then
  script_kv "QQ Webhook" "$(cat "${PUBLIC_URL_FILE}")"
fi
