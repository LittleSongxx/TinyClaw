#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

if [ -z "$1" ]; then
  echo "❌ version is necessary: ./scripts/docker.sh v1.0.0"
  exit 1
fi

VERSION=$1
IMAGE_NAME=tinyclaw

# Docker Hub 配置
DOCKER_HUB_USER=${DOCKER_HUB_USER:-littlesongxx}
DOCKER_HUB_REPO=${DOCKER_HUB_USER}/${IMAGE_NAME}
ALIYUN_REGISTRY=${ALIYUN_REGISTRY:-}

PLATFORMS="linux/amd64,linux/arm64"

echo "🚀 create multi-platform image..."
BUILD_ARGS=(
  --platform "${PLATFORMS}"
  -t "${DOCKER_HUB_REPO}:${VERSION}"
  -t "${DOCKER_HUB_REPO}:latest"
)

if [ -n "${ALIYUN_REGISTRY}" ]; then
  ALIYUN_REPO="${ALIYUN_REGISTRY}/${DOCKER_HUB_USER}/${IMAGE_NAME}"
  BUILD_ARGS+=(
    -t "${ALIYUN_REPO}:${VERSION}"
    -t "${ALIYUN_REPO}:latest"
  )
fi

docker buildx build -f deploy/docker/Dockerfile "${BUILD_ARGS[@]}" --push .


# Example:
# ALIYUN_REGISTRY=registry.cn-hangzhou.aliyuncs.com ./scripts/docker.sh v1.0.0

echo "✅ success"
