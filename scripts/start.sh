#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

cd "${REPO_ROOT}"

START_VERBOSE="${START_VERBOSE:-false}"
START_LOG_DIR="${LOG_DIR}/startup"
START_RUN_ID="${START_RUN_ID:-$(date +%Y%m%d-%H%M%S)}"

step() {
  printf '[start] %s\n' "$1"
}

step_done() {
  printf '[ok] %s\n' "$1"
}

step_fail() {
  printf '[error] %s\n' "$1" >&2
}

run_step() {
  local title="$1"
  local logfile="$2"
  shift 2

  step "${title}"
  if [[ "${START_VERBOSE}" == "true" ]]; then
    if ! "$@"; then
      step_fail "${title}"
      echo "Tip: inspect Docker/App logs with ./scripts/status.sh or rerun with START_VERBOSE=true ./scripts/start.sh." >&2
      exit 1
    fi
  else
    if ! "$@" >"${logfile}" 2>&1; then
      step_fail "${title}"
      echo "Last 80 lines from ${logfile}:" >&2
      tail -n 80 "${logfile}" >&2 || true
      echo "Full log: ${logfile}" >&2
      echo "Tip: rerun with START_VERBOSE=true ./scripts/start.sh for full output." >&2
      exit 1
    fi
  fi
  step_done "${title}"
}

build_app_image() {
  BUILDKIT_PROGRESS=plain docker_compose build app
}

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Missing ${ENV_FILE}" >&2
  exit 1
fi

source_env_file "${ENV_FILE}"

mkdir -p "${DATA_DIR}" "${DATA_DIR}/knowledge" "${LOG_DIR}" "${START_LOG_DIR}"

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
  step "Reusing current runtime ports from ${RUNTIME_FILE}."
else
  HOST_HTTP_PORT="$(find_free_port "${HOST_HTTP_PORT:-${HOST_HTTP_PORT_START:-36060}}")"
  HOST_ADMIN_PORT="$(find_free_port "${HOST_ADMIN_PORT:-${HOST_ADMIN_PORT_START:-18080}}")"
fi

export HOST_HTTP_PORT HOST_ADMIN_PORT

cat > "${RUNTIME_FILE}" <<EOF
HOST_HTTP_PORT=${HOST_HTTP_PORT}
HOST_ADMIN_PORT=${HOST_ADMIN_PORT}
EOF

deps_log="${START_LOG_DIR}/${START_RUN_ID}-deps.log"
build_log="${START_LOG_DIR}/${START_RUN_ID}-build.log"
app_log="${START_LOG_DIR}/${START_RUN_ID}-app.log"
cloudflared_log="${START_LOG_DIR}/${START_RUN_ID}-cloudflared.log"

run_step "Starting dependency containers" "${deps_log}" \
  docker_compose up -d --remove-orphans postgres redis hf-embeddings etcd minio milvus playwright-mcp

run_step "Building TinyClaw app image (first run may take several minutes)" "${build_log}" \
  build_app_image

run_step "Recreating TinyClaw app container" "${app_log}" \
  docker_compose up -d --no-deps --force-recreate app

if [[ "${ENABLE_CLOUDFLARED}" == "true" ]]; then
  run_step "Starting Cloudflared tunnel" "${cloudflared_log}" \
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

step "Waiting for TinyClaw HTTP"
if ! wait_for_http "http://127.0.0.1:${HOST_HTTP_PORT}/pong" "TinyClaw HTTP /pong" 90; then
  app_wait_log="${START_LOG_DIR}/${START_RUN_ID}-app-wait.log"
  docker_compose logs --tail=120 app >"${app_wait_log}" 2>&1 || true
  echo "Recent app logs:" >&2
  tail -n 80 "${app_wait_log}" >&2 || true
  echo "Full log: ${app_wait_log}" >&2
  exit 1
fi
step_done "TinyClaw HTTP is ready"

step "Waiting for TinyClaw Admin"
if ! wait_for_http "http://127.0.0.1:${HOST_ADMIN_PORT}/" "TinyClaw Admin" 90; then
  app_wait_log="${START_LOG_DIR}/${START_RUN_ID}-app-wait.log"
  docker_compose logs --tail=120 app >"${app_wait_log}" 2>&1 || true
  echo "Recent app logs:" >&2
  tail -n 80 "${app_wait_log}" >&2 || true
  echo "Full log: ${app_wait_log}" >&2
  exit 1
fi
step_done "TinyClaw Admin is ready"

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
printf 'Startup logs: %s\n' "${START_LOG_DIR}"
if [[ -n "${tunnel_url}" ]]; then
  printf 'QQ Webhook: %s/qq\n' "${tunnel_url}"
fi
