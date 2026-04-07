#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

cd "${REPO_ROOT}"

source_env_file "${ENV_FILE}"
docker_compose ps

if [[ -f "${RUNTIME_FILE}" ]]; then
  source_env_file "${RUNTIME_FILE}"
  echo
  cat "${RUNTIME_FILE}"
fi

echo
if [[ -n "${HOST_HTTP_PORT:-}" ]]; then
  printf 'HTTP_URL=http://127.0.0.1:%s\n' "${HOST_HTTP_PORT}"
fi
if [[ -n "${HOST_ADMIN_PORT:-}" ]]; then
  printf 'ADMIN_URL=http://127.0.0.1:%s\n' "${HOST_ADMIN_PORT}"
fi
printf 'AUTO_START=enabled (restart: unless-stopped)\n'
printf 'VERIFY_SCRIPT=./scripts/verify.sh\n'

if [[ -f "${PUBLIC_URL_FILE}" ]]; then
  echo
  printf 'QQ_WEBHOOK=%s\n' "$(cat "${PUBLIC_URL_FILE}")"
fi
