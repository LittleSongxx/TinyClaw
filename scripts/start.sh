#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

cd "${REPO_ROOT}"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Missing ${ENV_FILE}" >&2
  exit 1
fi

source_env_file "${ENV_FILE}"

mkdir -p "${DATA_DIR}" "${DATA_DIR}/knowledge" "${LOG_DIR}"

ENABLE_CLOUDFLARED="${ENABLE_CLOUDFLARED:-false}"

find_free_port() {
  local port="$1"
  while ss -ltnH "( sport = :${port} )" 2>/dev/null | grep -q .; do
    port=$((port + 1))
  done
  echo "${port}"
}

if [[ -f "${RUNTIME_FILE}" ]]; then
  source_env_file "${RUNTIME_FILE}"
fi

HOST_HTTP_PORT="${HOST_HTTP_PORT:-}"
HOST_ADMIN_PORT="${HOST_ADMIN_PORT:-}"
existing_app_id="$(docker_compose ps -q app 2>/dev/null || true)"
reuse_runtime_ports=false
if [[ -n "${existing_app_id}" && -n "${HOST_HTTP_PORT}" && -n "${HOST_ADMIN_PORT}" ]]; then
  reuse_runtime_ports=true
fi

if [[ "${reuse_runtime_ports}" == "true" ]]; then
  echo "Reusing current runtime ports from ${RUNTIME_FILE}."
else
  HOST_HTTP_PORT="$(find_free_port "${HOST_HTTP_PORT:-${HOST_HTTP_PORT_START:-36060}}")"
  HOST_ADMIN_PORT="$(find_free_port "${HOST_ADMIN_PORT:-${HOST_ADMIN_PORT_START:-18080}}")"
fi

export HOST_HTTP_PORT HOST_ADMIN_PORT

cat > "${RUNTIME_FILE}" <<EOF
HOST_HTTP_PORT=${HOST_HTTP_PORT}
HOST_ADMIN_PORT=${HOST_ADMIN_PORT}
EOF

docker_compose up -d --remove-orphans postgres redis hf-embeddings etcd minio milvus playwright-mcp
docker_compose build app
docker_compose up -d --no-deps --force-recreate app

if [[ "${ENABLE_CLOUDFLARED}" == "true" ]]; then
  docker_compose up -d cloudflared
fi

wait_for_http() {
  local url="$1"
  local name="$2"
  local max_attempts="${3:-60}"
  local attempt

  for attempt in $(seq 1 "${max_attempts}"); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done

  echo "Timed out waiting for ${name}: ${url}" >&2
  return 1
}

wait_for_http "http://127.0.0.1:${HOST_HTTP_PORT}/pong" "TinyClaw HTTP /pong" 90
wait_for_http "http://127.0.0.1:${HOST_ADMIN_PORT}/" "TinyClaw Admin" 90

tunnel_url=""
rm -f "${PUBLIC_URL_FILE}"
if [[ "${ENABLE_CLOUDFLARED}" == "true" ]]; then
  for _ in $(seq 1 30); do
    tunnel_url="$(docker_compose logs cloudflared 2>/dev/null | grep -Eo 'https://[-a-z0-9]+\.trycloudflare\.com' | tail -n 1 || true)"
    if [[ -n "${tunnel_url}" ]]; then
      break
    fi
    sleep 2
  done

  if [[ -n "${tunnel_url}" ]]; then
    printf '%s/qq\n' "${tunnel_url}" > "${PUBLIC_URL_FILE}"
  fi
fi

printf 'TinyClaw HTTP: http://127.0.0.1:%s\n' "${HOST_HTTP_PORT}"
printf 'TinyClaw Admin: http://127.0.0.1:%s\n' "${HOST_ADMIN_PORT}"
printf 'Auto-start: enabled (Docker restart policy: unless-stopped)\n'
printf 'Verify: ./scripts/verify.sh\n'
if [[ -n "${tunnel_url}" ]]; then
  printf 'QQ Webhook: %s/qq\n' "${tunnel_url}"
fi
