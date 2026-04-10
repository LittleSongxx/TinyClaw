#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

SCRIPT_HINT_CMD="./scripts/docker.sh <version>"
SCRIPT_RUN_ID="${SCRIPT_RUN_ID:-$(date +%Y%m%d-%H%M%S)}"

cd "${REPO_ROOT}"
ensure_script_log_dir

if [[ -z "${1:-}" ]]; then
  script_error "Version is required."
  echo "Usage: ./scripts/docker.sh v1.0.0" >&2
  exit 1
fi

VERSION="$1"
IMAGE_NAME="tinyclaw"
DOCKER_HUB_USER="${DOCKER_HUB_USER:-littlesongxx}"
DOCKER_HUB_REPO="${DOCKER_HUB_USER}/${IMAGE_NAME}"
ALIYUN_REGISTRY="${ALIYUN_REGISTRY:-}"
PLATFORMS="linux/amd64,linux/arm64"
BUILD_LOG="${SCRIPT_LOG_DIR}/${SCRIPT_RUN_ID}-docker-build.log"

script_section "Docker Release"
script_kv "Version" "${VERSION}"
script_kv "Platforms" "${PLATFORMS}"
script_kv "Docker Hub" "${DOCKER_HUB_REPO}"

BUILD_ARGS=(
  --platform "${PLATFORMS}"
  -f deploy/docker/Dockerfile
  -t "${DOCKER_HUB_REPO}:${VERSION}"
  -t "${DOCKER_HUB_REPO}:latest"
)

if [[ -n "${ALIYUN_REGISTRY}" ]]; then
  ALIYUN_REPO="${ALIYUN_REGISTRY}/${DOCKER_HUB_USER}/${IMAGE_NAME}"
  BUILD_ARGS+=(
    -t "${ALIYUN_REPO}:${VERSION}"
    -t "${ALIYUN_REPO}:latest"
  )
  script_kv "Aliyun" "${ALIYUN_REPO}"
fi

run_with_log "Building and pushing multi-platform image" "${BUILD_LOG}" \
  docker buildx build "${BUILD_ARGS[@]}" --push .

script_section "Done"
script_ok "Docker image release completed"
script_kv "Image tag" "${DOCKER_HUB_REPO}:${VERSION}"
script_kv "Latest tag" "${DOCKER_HUB_REPO}:latest"
script_kv "Build log" "${BUILD_LOG}"
