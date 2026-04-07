#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

cd "${REPO_ROOT}"

source_env_file "${ENV_FILE}"
docker_compose ps

if [[ -f "${RUNTIME_FILE}" ]]; then
  echo
  cat "${RUNTIME_FILE}"
fi

echo
printf 'AUTO_START=enabled (restart: unless-stopped)\n'

if [[ -f "${PUBLIC_URL_FILE}" ]]; then
  echo
  printf 'QQ_WEBHOOK=%s\n' "$(cat "${PUBLIC_URL_FILE}")"
fi
