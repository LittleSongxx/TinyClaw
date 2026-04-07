#!/usr/bin/env bash
set -euo pipefail

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

if [[ "${EMBEDDING_TYPE:-}" == "huggingface" && -n "${EMBEDDING_BASE_URL:-}" ]]; then
  wait_for_http "HF embeddings" "${EMBEDDING_BASE_URL%/}/health" "200"
fi

if [[ "${VECTOR_DB_TYPE:-}" == "milvus" ]]; then
  wait_for_http "Milvus" "http://milvus:9091/healthz" "200"
fi

if [[ "${USE_TOOLS:-false}" == "true" ]]; then
  wait_for_http "Playwright MCP" "http://playwright-mcp:8931/mcp" "200,400,405"
fi

exec /app/TinyClaw
