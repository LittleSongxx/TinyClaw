#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

SCRIPT_HINT_CMD="./scripts/release.sh"
SCRIPT_RUN_ID="${SCRIPT_RUN_ID:-$(date +%Y%m%d-%H%M%S)}"

BUILD_ROOT="${REPO_ROOT}/build"
OUTPUT_DIR="${BUILD_ROOT}/output"
RELEASE_DIR="${BUILD_ROOT}/release"

cd "${REPO_ROOT}"
ensure_script_log_dir

script_section "Preparing Release Workspace"
rm -rf "${OUTPUT_DIR}" "${RELEASE_DIR}"
mkdir -p "${OUTPUT_DIR}" "${RELEASE_DIR}"
script_ok "Release workspace is ready"

build_admin_local() {
  local os="$1"
  local arch="$2"
  local log_file="${SCRIPT_LOG_DIR}/${SCRIPT_RUN_ID}-release-admin-${os}-${arch}.log"

  run_with_log "Building TinyClawAdmin for ${os}/${arch}" "${log_file}" \
    env GOOS="${os}" GOARCH="${arch}" CGO_ENABLED=1 go build -o "${OUTPUT_DIR}/TinyClawAdmin" ./admin
}

compile_and_package_local() {
  local os="$1"
  local arch="$2"
  local build_log="${SCRIPT_LOG_DIR}/${SCRIPT_RUN_ID}-release-${os}-${arch}.log"
  local package_log="${SCRIPT_LOG_DIR}/${SCRIPT_RUN_ID}-release-package-${os}-${arch}.log"
  local release_name="TinyClaw-${os}-${arch}.tar.gz"

  script_section "Packaging ${os}/${arch}"
  run_with_log "Building TinyClaw for ${os}/${arch}" "${build_log}" \
    env GOOS="${os}" GOARCH="${arch}" CGO_ENABLED=1 go build -o "${OUTPUT_DIR}/TinyClaw" ./cmd/tinyclaw

  build_admin_local "${os}" "${arch}"

  mkdir -p "${OUTPUT_DIR}/conf" "${OUTPUT_DIR}/data"
  cp -r ./conf/i18n "${OUTPUT_DIR}/conf/"
  cp -r ./conf/mcp "${OUTPUT_DIR}/conf/"
  cp -r ./conf/img "${OUTPUT_DIR}/conf/"

  run_with_log "Creating ${release_name}" "${package_log}" \
    tar zcf "${RELEASE_DIR}/${release_name}" -C "${OUTPUT_DIR}" .

  script_ok "Packaged ${release_name}"
  script_kv "Archive" "${RELEASE_DIR}/${release_name}"
}

compile_and_package_local darwin amd64
compile_and_package_local darwin arm64

rm -rf "${OUTPUT_DIR}"

script_section "Done"
script_ok "Release packaging completed"
script_kv "Release dir" "${RELEASE_DIR}"
