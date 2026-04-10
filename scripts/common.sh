#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${REPO_ROOT}/deploy/docker"
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.yml"
ENV_FILE="${DEPLOY_DIR}/.env"
RUNTIME_FILE="${DEPLOY_DIR}/.env.runtime"
PUBLIC_URL_FILE="${DEPLOY_DIR}/public_url.txt"
DATA_DIR="${REPO_ROOT}/data"
LOG_DIR="${REPO_ROOT}/log"
WINDOWS_DOCKER="/mnt/c/Program Files/Docker/Docker/resources/bin/docker.exe"
SCRIPT_LOG_DIR="${LOG_DIR}/scripts"

source_env_file() {
  local env_file="$1"
  if [[ -f "${env_file}" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "${env_file}"
    set +a
  fi
}

to_docker_path() {
  local path="$1"
  wslpath -w "${path}" | tr '\\' '/'
}

resolve_docker_backend() {
  DOCKER_BIN="docker"
  DOCKER_PATH_MODE="linux"

  if docker ps >/dev/null 2>&1; then
    return
  fi

  if [[ -x "${WINDOWS_DOCKER}" ]] && "${WINDOWS_DOCKER}" version >/dev/null 2>&1; then
    DOCKER_BIN="${WINDOWS_DOCKER}"
    DOCKER_PATH_MODE="windows"
    return
  fi

  echo "Docker daemon is unavailable. Start Docker Desktop and ensure WSL integration is enabled." >&2
  exit 1
}

ensure_script_log_dir() {
  mkdir -p "${LOG_DIR}" "${SCRIPT_LOG_DIR}"
}

script_info() {
  printf '[info] %s\n' "$1"
}

script_ok() {
  printf '[ok] %s\n' "$1"
}

script_warn() {
  printf '[warn] %s\n' "$1" >&2
}

script_error() {
  printf '[error] %s\n' "$1" >&2
}

script_section() {
  printf '\n== %s ==\n' "$1"
}

script_kv() {
  printf '  %-14s %s\n' "$1" "$2"
}

run_with_log() {
  local title="$1"
  local logfile="$2"
  shift 2

  ensure_script_log_dir
  script_info "${title}"

  if [[ "${SCRIPT_VERBOSE:-false}" == "true" ]]; then
    if ! "$@"; then
      script_error "${title}"
      echo "Tip: rerun with SCRIPT_VERBOSE=true ${SCRIPT_HINT_CMD:-$0}" >&2
      return 1
    fi
  else
    if ! "$@" >"${logfile}" 2>&1; then
      script_error "${title}"
      echo "Last 60 lines from ${logfile}:" >&2
      tail -n 60 "${logfile}" >&2 || true
      echo "Full log: ${logfile}" >&2
      echo "Tip: rerun with SCRIPT_VERBOSE=true ${SCRIPT_HINT_CMD:-$0}" >&2
      return 1
    fi
  fi

  script_ok "${title}"
}

compose_status_table() {
  if ! docker_compose ps --format 'table {{.Service}}\t{{.Status}}'; then
    docker_compose ps
  fi
}

docker_compose() {
  local profile_args=()

  if [[ "${ENABLE_CLOUDFLARED:-false}" == "true" ]]; then
    profile_args+=(--profile cloudflared)
  fi

  if [[ "${ENABLE_KNOWLEDGE:-false}" == "true" ]]; then
    profile_args+=(--profile knowledge)
  fi

  if [[ "${ENABLE_FULL_STACK:-false}" == "true" ]]; then
    profile_args+=(--profile full)
  fi

  resolve_docker_backend

  if [[ "${DOCKER_PATH_MODE}" == "windows" ]]; then
    local compose_args=(
      -f "$(to_docker_path "${COMPOSE_FILE}")"
      --project-directory "$(to_docker_path "${DEPLOY_DIR}")"
      --env-file "$(to_docker_path "${ENV_FILE}")"
    )

    if [[ -f "${RUNTIME_FILE}" ]]; then
      compose_args+=(--env-file "$(to_docker_path "${RUNTIME_FILE}")")
    fi

  "${DOCKER_BIN}" compose \
      "${compose_args[@]}" \
      "${profile_args[@]}" \
      "$@"
    return
  fi

  local compose_args=(
    -f "${COMPOSE_FILE}"
    --project-directory "${DEPLOY_DIR}"
    --env-file "${ENV_FILE}"
  )

  if [[ -f "${RUNTIME_FILE}" ]]; then
    compose_args+=(--env-file "${RUNTIME_FILE}")
  fi

  "${DOCKER_BIN}" compose \
    "${compose_args[@]}" \
    "${profile_args[@]}" \
    "$@"
}
