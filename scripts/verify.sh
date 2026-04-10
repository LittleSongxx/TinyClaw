#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

SCRIPT_HINT_CMD="./scripts/verify.sh [--full]"

cd "${REPO_ROOT}"

source_env_file "${ENV_FILE}"
if [[ -f "${RUNTIME_FILE}" ]]; then
  source_env_file "${RUNTIME_FILE}"
fi

VERIFY_FULL=false
if [[ "${1:-}" == "--full" ]]; then
  VERIFY_FULL=true
fi

HOST_HTTP_PORT="${HOST_HTTP_PORT:-${HOST_HTTP_PORT_START:-36060}}"
HOST_ADMIN_PORT="${HOST_ADMIN_PORT:-${HOST_ADMIN_PORT_START:-18080}}"
HTTP_BASE="http://127.0.0.1:${HOST_HTTP_PORT}"
ADMIN_BASE="http://127.0.0.1:${HOST_ADMIN_PORT}"

VERIFY_RUN_ID="${VERIFY_RUN_ID:-$(date +%s)}"
VERIFY_MARKER="${VERIFY_MARKER:-tinyclaw-verify-marker-${VERIFY_RUN_ID}}"
VERIFY_DOC_NAME="${VERIFY_DOC_NAME:-tinyclaw-live-verify-${VERIFY_RUN_ID}.txt}"
VERIFY_DOC_CONTENT="${VERIFY_DOC_CONTENT:-TinyClaw live verification marker: ${VERIFY_MARKER}. TinyClaw uses PostgreSQL pgvector, Redis, and MinIO as the new knowledge stack.}"
VERIFY_DOC_QUERY="${VERIFY_DOC_QUERY:-${VERIFY_MARKER}}"
VERIFY_USER_BASE="${VERIFY_USER_BASE:-$((100000 + (VERIFY_RUN_ID % 900000) * 10))}"
VERIFY_TASK_USER_ID="${VERIFY_TASK_USER_ID:-$((VERIFY_USER_BASE + 1))}"
VERIFY_MCP_USER_ID="${VERIFY_MCP_USER_ID:-$((VERIFY_USER_BASE + 2))}"

request_json() {
  local method="$1"
  local url="$2"
  local data="${3:-}"
  local response

  if [[ -n "${data}" ]]; then
    response="$(curl -fsS -X "${method}" -H 'Content-Type: application/json' -d "${data}" "${url}")"
  else
    response="$(curl -fsS -X "${method}" "${url}")"
  fi

  if [[ "${response}" != *'"code":0'* ]]; then
    echo "API verification failed: ${url}" >&2
    echo "${response}" >&2
    return 1
  fi

  printf '%s' "${response}"
}

verify_step() {
  script_info "$1"
}

verify_ok() {
  script_ok "$1"
}

script_section "Verification"
script_kv "HTTP" "${HTTP_BASE}"
script_kv "Admin" "${ADMIN_BASE}"
script_kv "Mode" "$([[ "${VERIFY_FULL}" == "true" ]] && echo "full" || echo "basic")"

verify_step "Checking /pong"
pong_response="$(curl -fsS "${HTTP_BASE}/pong")"
grep -q "pong" <<<"${pong_response}"
verify_ok "HTTP /pong"

verify_step "Checking /metrics"
metrics_response="$(curl -fsS "${HTTP_BASE}/metrics")"
grep -q "^go_goroutines " <<<"${metrics_response}"
verify_ok "HTTP /metrics"

verify_step "Checking Admin root"
admin_response="$(curl -fsS "${ADMIN_BASE}/")"
grep -Eqi 'TinyClaw|<html' <<<"${admin_response}"
verify_ok "Admin root"

verify_step "Checking agent run API"
request_json GET "${HTTP_BASE}/run/list?page=1&page_size=5" >/dev/null
verify_ok "Agent run API"

verify_step "Checking knowledge collection API"
request_json GET "${HTTP_BASE}/knowledge/collections/list" >/dev/null
verify_ok "Knowledge collection API"

verify_step "Creating verification document"
create_payload="$(printf '{"file_name":"%s","content":"%s"}' "${VERIFY_DOC_NAME}" "${VERIFY_DOC_CONTENT}" | sed 's/\\/\\\\/g')"
request_json POST "${HTTP_BASE}/knowledge/documents/create" "${create_payload}" >/dev/null
verify_ok "Verification document created"

verify_step "Waiting for ingestion job"
job_ok=false
for _ in $(seq 1 60); do
  jobs_response="$(request_json GET "${HTTP_BASE}/knowledge/jobs/list?page=1&page_size=20" || true)"
  if [[ "${jobs_response}" == *"\"document_name\":\"${VERIFY_DOC_NAME}\""* && "${jobs_response}" == *'"status":"succeeded"'* ]]; then
    job_ok=true
    break
  fi
  if [[ "${jobs_response}" == *"\"document_name\":\"${VERIFY_DOC_NAME}\""* && "${jobs_response}" == *'"status":"failed"'* ]]; then
    echo "${jobs_response}" >&2
    break
  fi
  sleep 2
done

if [[ "${job_ok}" != "true" ]]; then
    script_error "Knowledge ingestion job did not finish successfully for ${VERIFY_DOC_NAME}"
  exit 1
fi
verify_ok "Ingestion job succeeded"

verify_step "Running retrieval debug"
debug_payload="$(printf '{"query":"%s"}' "${VERIFY_DOC_QUERY}" | sed 's/\\/\\\\/g')"
debug_ok=false
debug_response=""
for _ in $(seq 1 30); do
  debug_response="$(request_json POST "${HTTP_BASE}/knowledge/retrieval/debug" "${debug_payload}")"
  if [[ "${debug_response}" == *"\"document_name\":\"${VERIFY_DOC_NAME}\""* && "${debug_response}" == *"${VERIFY_MARKER}"* ]]; then
    debug_ok=true
    break
  fi
  sleep 2
done

if [[ "${debug_ok}" != "true" ]]; then
  script_error "Retrieval debug response does not include the verification document"
  echo "${debug_response}" >&2
  exit 1
fi
verify_ok "Retrieval debug"

if [[ "${VERIFY_FULL}" == "true" ]]; then
  verify_step "Running /task live verification"
  task_output="$(curl -fsS -N --get \
    --data-urlencode "user_id=${VERIFY_TASK_USER_ID}" \
    --data-urlencode "prompt=/task 请只回复四个字：任务完成" \
    "${HTTP_BASE}/communicate")"
  if [[ "${task_output}" != *"任务完成"* ]]; then
    script_error "Task verification failed"
    echo "${task_output}" >&2
    exit 1
  fi
  verify_ok "/task live verification"

  verify_step "Running /mcp live verification"
  mcp_output="$(curl -fsS -N --get \
    --data-urlencode "user_id=${VERIFY_MCP_USER_ID}" \
    --data-urlencode "prompt=/mcp 打开 https://example.com 并只返回页面标题" \
    "${HTTP_BASE}/communicate")"
  if [[ "${mcp_output}" != *"Example Domain"* ]]; then
    script_error "MCP verification failed"
    echo "${mcp_output}" >&2
    exit 1
  fi
  verify_ok "/mcp live verification"

  verify_step "Verifying replay API"
  task_run_list="$(request_json GET "${HTTP_BASE}/run/list?page=1&page_size=5&mode=task&user_id=${VERIFY_TASK_USER_ID}")"
  task_run_id="$(printf '%s' "${task_run_list}" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p' | head -n 1)"
  if [[ -z "${task_run_id}" ]]; then
    script_error "Could not resolve task run id for replay"
    echo "${task_run_list}" >&2
    exit 1
  fi

  replay_response="$(curl -fsS -X POST "${HTTP_BASE}/run/replay" -d "id=${task_run_id}")"
  if [[ "${replay_response}" != *'"code":0'* || "${replay_response}" != *"\"replay_of\":${task_run_id}"* ]]; then
    script_error "Replay verification failed"
    echo "${replay_response}" >&2
    exit 1
  fi
  verify_ok "Replay API"
fi

script_section "Done"
script_ok "Verification succeeded"
script_kv "HTTP" "${HTTP_BASE}"
script_kv "Admin" "${ADMIN_BASE}"
