#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

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

echo "[1/8] Checking /pong"
pong_response="$(curl -fsS "${HTTP_BASE}/pong")"
grep -q "pong" <<<"${pong_response}"

echo "[2/8] Checking /metrics"
metrics_response="$(curl -fsS "${HTTP_BASE}/metrics")"
grep -q "^go_goroutines " <<<"${metrics_response}"

echo "[3/8] Checking Admin root"
admin_response="$(curl -fsS "${ADMIN_BASE}/")"
grep -Eqi 'TinyClaw|<html' <<<"${admin_response}"

echo "[4/8] Checking Agent run API"
request_json GET "${HTTP_BASE}/run/list?page=1&page_size=5" >/dev/null

echo "[5/8] Checking RAG collection API"
request_json GET "${HTTP_BASE}/rag/collections/list" >/dev/null

echo "[6/8] Creating verification document"
create_payload="$(printf '{"file_name":"%s","content":"%s"}' "${VERIFY_DOC_NAME}" "${VERIFY_DOC_CONTENT}" | sed 's/\\/\\\\/g')"
request_json POST "${HTTP_BASE}/rag/documents/create" "${create_payload}" >/dev/null

echo "[7/8] Waiting for ingestion job to finish"
job_ok=false
for _ in $(seq 1 60); do
  jobs_response="$(request_json GET "${HTTP_BASE}/rag/jobs/list?page=1&page_size=20" || true)"
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
  echo "RAG ingestion job did not finish successfully for ${VERIFY_DOC_NAME}" >&2
  exit 1
fi

echo "[8/8] Running retrieval debug"
debug_payload="$(printf '{"query":"%s"}' "${VERIFY_DOC_QUERY}" | sed 's/\\/\\\\/g')"
debug_ok=false
debug_response=""
for _ in $(seq 1 30); do
  debug_response="$(request_json POST "${HTTP_BASE}/rag/retrieval/debug" "${debug_payload}")"
  if [[ "${debug_response}" == *"\"document_name\":\"${VERIFY_DOC_NAME}\""* && "${debug_response}" == *"${VERIFY_MARKER}"* ]]; then
    debug_ok=true
    break
  fi
  sleep 2
done

if [[ "${debug_ok}" != "true" ]]; then
  echo "Retrieval debug response does not include the verification document" >&2
  echo "${debug_response}" >&2
  exit 1
fi

if [[ "${VERIFY_FULL}" == "true" ]]; then
  echo "[9/11] Running /task live verification"
  task_output="$(curl -fsS -N --get \
    --data-urlencode "user_id=${VERIFY_TASK_USER_ID}" \
    --data-urlencode "prompt=/task 请只回复四个字：任务完成" \
    "${HTTP_BASE}/communicate")"
  if [[ "${task_output}" != *"任务完成"* ]]; then
    echo "Task verification failed" >&2
    echo "${task_output}" >&2
    exit 1
  fi

  echo "[10/11] Running /mcp live verification"
  mcp_output="$(curl -fsS -N --get \
    --data-urlencode "user_id=${VERIFY_MCP_USER_ID}" \
    --data-urlencode "prompt=/mcp 打开 https://example.com 并只返回页面标题" \
    "${HTTP_BASE}/communicate")"
  if [[ "${mcp_output}" != *"Example Domain"* ]]; then
    echo "MCP verification failed" >&2
    echo "${mcp_output}" >&2
    exit 1
  fi

  echo "[11/11] Verifying replay API"
  task_run_list="$(request_json GET "${HTTP_BASE}/run/list?page=1&page_size=5&mode=task&user_id=${VERIFY_TASK_USER_ID}")"
  task_run_id="$(printf '%s' "${task_run_list}" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p' | head -n 1)"
  if [[ -z "${task_run_id}" ]]; then
    echo "Could not resolve task run id for replay" >&2
    echo "${task_run_list}" >&2
    exit 1
  fi

  replay_response="$(curl -fsS -X POST "${HTTP_BASE}/run/replay" -d "id=${task_run_id}")"
  if [[ "${replay_response}" != *'"code":0'* || "${replay_response}" != *"\"replay_of\":${task_run_id}"* ]]; then
    echo "Replay verification failed" >&2
    echo "${replay_response}" >&2
    exit 1
  fi
fi

echo
printf 'Verification succeeded.\n'
printf 'HTTP: %s\n' "${HTTP_BASE}"
printf 'Admin: %s\n' "${ADMIN_BASE}"
