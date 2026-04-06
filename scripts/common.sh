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

docker_compose() {
  local profile_args=()

  if [[ "${ENABLE_CLOUDFLARED:-false}" == "true" ]]; then
    profile_args+=(--profile cloudflared)
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
