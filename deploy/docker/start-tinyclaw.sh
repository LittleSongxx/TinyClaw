#!/usr/bin/env bash
set -euo pipefail

log() {
  printf '[boot] %s\n' "$1"
}

mkdir -p /app/data/mcp/memory /app/data/mcp/arxiv /app/data/sessions

wait_for_http() {
  local name="$1"
  local url="$2"
  local allowed_codes="$3"
  local attempts="${4:-120}"

  for _ in $(seq 1 "${attempts}"); do
    local status
    status="$(http_status "${url}")"
    if [[ ",${allowed_codes}," == *",${status},"* ]]; then
      return 0
    fi
    sleep 1
  done

  echo "Timed out waiting for ${name} at ${url}" >&2
  return 1
}

wait_for_tcp() {
  local name="$1"
  local host="$2"
  local port="$3"
  local attempts="${4:-120}"

  for _ in $(seq 1 "${attempts}"); do
    if exec 3<>"/dev/tcp/${host}/${port}" 2>/dev/null; then
      exec 3<&-
      exec 3>&-
      return 0
    fi
    sleep 1
  done

  echo "Timed out waiting for ${name} at ${host}:${port}" >&2
  return 1
}

split_host_port() {
  local value="$1"
  local default_port="$2"

  value="${value#*://}"
  value="${value%%/*}"
  value="${value##*@}"

  local host="${value}"
  local port="${default_port}"
  if [[ "${value}" == *:* ]]; then
    host="${value%%:*}"
    port="${value##*:}"
  fi

  printf '%s %s\n' "${host}" "${port}"
}

http_status() {
  local url="$1"

  if command -v curl >/dev/null 2>&1; then
    curl -s -o /dev/null -w '%{http_code}' "${url}" || true
    return
  fi

  local rest hostport path host port status_line
  rest="${url#http://}"
  hostport="${rest%%/*}"
  if [[ "${rest}" == "${hostport}" ]]; then
    path="/"
  else
    path="/${rest#*/}"
  fi

  host="${hostport%%:*}"
  port="${hostport#*:}"
  if [[ "${host}" == "${port}" ]]; then
    port="80"
  fi

  exec 3<>"/dev/tcp/${host}/${port}" || {
    echo "000"
    return
  }

  printf 'GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n' "${path}" "${host}" >&3
  IFS=$'\r' read -r status_line <&3 || true
  exec 3<&-
  exec 3>&-

  set -- ${status_line}
  echo "${2:-000}"
}

if [[ "${ENABLE_KNOWLEDGE:-false}" == "true" ]]; then
  if [[ -n "${POSTGRES_DSN:-}" ]]; then
    read -r postgres_host postgres_port < <(split_host_port "${POSTGRES_DSN}" "5432")
    log "Waiting for PostgreSQL"
    wait_for_tcp "PostgreSQL" "${postgres_host}" "${postgres_port}"
  fi

  if [[ -n "${REDIS_ADDR:-}" ]]; then
    read -r redis_host redis_port < <(split_host_port "${REDIS_ADDR}" "6379")
    log "Waiting for Redis"
    wait_for_tcp "Redis" "${redis_host}" "${redis_port}"
  fi

  if [[ -n "${MINIO_ENDPOINT:-}" ]]; then
    read -r minio_host minio_port < <(split_host_port "${MINIO_ENDPOINT}" "9000")
    minio_scheme="http"
    if [[ "${MINIO_USE_SSL:-false}" == "true" ]]; then
      minio_scheme="https"
    fi
    log "Waiting for MinIO"
    wait_for_http "MinIO" "${minio_scheme}://${minio_host}:${minio_port}/minio/health/live" "200"
  fi

  if [[ "${EMBEDDING_TYPE:-}" == "huggingface" && -n "${EMBEDDING_BASE_URL:-}" ]]; then
    log "Waiting for HF embeddings"
    wait_for_http "HF embeddings" "${EMBEDDING_BASE_URL%/}/health" "200"
  fi
fi

if [[ "${USE_TOOLS:-false}" == "true" ]]; then
  log "Waiting for Playwright MCP"
  wait_for_http "Playwright MCP" "http://playwright-mcp:8931/mcp" "200,400,405"
fi

log "Starting TinyClaw"
exec /app/TinyClaw
