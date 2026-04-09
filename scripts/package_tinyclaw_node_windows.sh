#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

SCRIPT_HINT_CMD="./scripts/package_tinyclaw_node_windows.sh [amd64|arm64]"
SCRIPT_RUN_ID="${SCRIPT_RUN_ID:-$(date +%Y%m%d-%H%M%S)}"

BUILD_ROOT="${REPO_ROOT}/build"
STAGE_ROOT="${BUILD_ROOT}/tinyclaw-node-windows"
RELEASE_DIR="${BUILD_ROOT}/release"
ARCH="${1:-amd64}"
PACKAGE_NAME="TinyClawNode-windows-${ARCH}"
PACKAGE_DIR="${STAGE_ROOT}/${PACKAGE_NAME}"
BUILD_LOG="${SCRIPT_LOG_DIR}/${SCRIPT_RUN_ID}-package-node-build.log"
ZIP_LOG="${SCRIPT_LOG_DIR}/${SCRIPT_RUN_ID}-package-node-zip.log"
INSTALLER_LOG="${SCRIPT_LOG_DIR}/${SCRIPT_RUN_ID}-package-node-installer.log"
NODE_VERSION_RAW="$(sed -n 's/^const nodeBinaryVersion = "v\(.*\)"/\1/p' cmd/tinyclaw-node/runtime.go | head -n 1)"
PRODUCT_VERSION="${NODE_VERSION_RAW:-0.2.0}"
SETUP_NAME="TinyClawNodeSetup.exe"

if [[ "${ARCH}" != "amd64" ]]; then
  SETUP_NAME="TinyClawNodeSetup-${ARCH}.exe"
fi

cd "${REPO_ROOT}"
ensure_script_log_dir

script_section "Windows Node Package"
script_kv "Target" "windows/${ARCH}"
script_kv "Output" "${RELEASE_DIR}/${PACKAGE_NAME}.zip"
script_kv "Installer" "${RELEASE_DIR}/${SETUP_NAME}"
script_kv "Version" "${PRODUCT_VERSION}"

rm -rf "${PACKAGE_DIR}"
mkdir -p "${PACKAGE_DIR}" "${RELEASE_DIR}"

run_with_log "Building tinyclaw-node.exe" "${BUILD_LOG}" \
  env GOOS=windows GOARCH="${ARCH}" CGO_ENABLED=0 go build -o "${PACKAGE_DIR}/tinyclaw-node.exe" ./cmd/tinyclaw-node

script_info "Copying Windows package assets"
cp \
  "${REPO_ROOT}/deploy/windows-node/configure-node.cmd" \
  "${REPO_ROOT}/deploy/windows-node/configure-node.ps1" \
  "${REPO_ROOT}/deploy/windows-node/configure-node.vbs" \
  "${REPO_ROOT}/deploy/windows-node/install-node.cmd" \
  "${REPO_ROOT}/deploy/windows-node/install-node.ps1" \
  "${REPO_ROOT}/deploy/windows-node/launch-node.vbs" \
  "${REPO_ROOT}/deploy/windows-node/config.template.json" \
  "${REPO_ROOT}/deploy/windows-node/README.md" \
  "${PACKAGE_DIR}/"
script_ok "Copied Windows package assets"

create_zip_package() {
  cd "${STAGE_ROOT}"
  if command -v zip >/dev/null 2>&1; then
    zip -qr "${RELEASE_DIR}/${PACKAGE_NAME}.zip" "${PACKAGE_NAME}"
    return
  fi
  if command -v python3 >/dev/null 2>&1; then
    python3 -m zipfile -c "${RELEASE_DIR}/${PACKAGE_NAME}.zip" "${PACKAGE_NAME}"
    return
  fi
  echo "zip or python3 is required to create ${PACKAGE_NAME}.zip" >&2
  return 1
}

run_with_log "Creating ${PACKAGE_NAME}.zip" "${ZIP_LOG}" create_zip_package

build_nsis_installer() {
  local installer_out="${RELEASE_DIR}/${SETUP_NAME}"
  local stage_path="${PACKAGE_DIR}"

  rm -f "${installer_out}"

  if command -v makensis >/dev/null 2>&1; then
    makensis \
      -DAPP_STAGE_DIR="${stage_path}" \
      -DOUTFILE="${installer_out}" \
      -DPRODUCT_VERSION="${PRODUCT_VERSION}" \
      -DTARGET_ARCH="${ARCH}" \
      "${REPO_ROOT}/deploy/windows-node/TinyClawNodeSetup.nsi"
    return
  fi

  local repo_in_container="/work"
  local container_stage="${repo_in_container}${stage_path#${REPO_ROOT}}"
  local container_out="${repo_in_container}${installer_out#${REPO_ROOT}}"
  docker run --rm \
    --user "$(id -u):$(id -g)" \
    -v "${REPO_ROOT}:${repo_in_container}" \
    -w "${repo_in_container}" \
    binfalse/nsis \
    -DAPP_STAGE_DIR="${container_stage}" \
    -DOUTFILE="${container_out}" \
    -DPRODUCT_VERSION="${PRODUCT_VERSION}" \
    -DTARGET_ARCH="${ARCH}" \
    "${repo_in_container}/deploy/windows-node/TinyClawNodeSetup.nsi"
}

run_with_log "Building ${SETUP_NAME}" "${INSTALLER_LOG}" build_nsis_installer

script_section "Done"
script_ok "Windows node package created"
script_kv "Archive" "${RELEASE_DIR}/${PACKAGE_NAME}.zip"
script_kv "Installer" "${RELEASE_DIR}/${SETUP_NAME}"
script_kv "Build log" "${BUILD_LOG}"
script_kv "Zip log" "${ZIP_LOG}"
script_kv "Installer log" "${INSTALLER_LOG}"
