#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILD_ROOT="${REPO_ROOT}/build"
OUTPUT_DIR="${BUILD_ROOT}/output"
RELEASE_DIR="${BUILD_ROOT}/release"

cd "${REPO_ROOT}"

# Clean up old files
rm -rf "${OUTPUT_DIR}" "${RELEASE_DIR}"
mkdir -p "${OUTPUT_DIR}" "${RELEASE_DIR}"

# Check if xgo is installed
if ! command -v xgo &> /dev/null; then
    echo "Installing xgo..."
    go install src.techknowlogick.com/xgo@latest
fi

# Build the admin binary (locally for the specified platform)
build_admin_local() {
    local os=$1
    local arch=$2
    local ext=""
    [[ "$os" == "windows" ]] && ext=".exe"

    local admin_output="TinyClawAdmin"

    echo "=============================="
    echo "Building admin [$os/$arch] using go build..."
    echo "=============================="
    xgo -out "$admin_output" -targets="$os/$arch" --hooksdir=./admin/shell ./admin
}

# Build main binary + package everything
compile_and_package() {
    local os=$1
    local arch=$2
    local ext=""
    [[ "$os" == "windows" ]] && ext=".exe"

    echo "=============================="
    echo "Building TinyClaw [$os/$arch] using xgo..."
    echo "=============================="

    # Build the main bot binary
    xgo -out TinyClaw -targets="$os/$arch" ./cmd/tinyclaw

    # Build admin binary
    build_admin_local $os $arch

    local bot_binary="TinyClaw$ext"
    local admin_binary="TinyClawAdmin$ext"
    local release_name="TinyClaw-${os}-${arch}.tar.gz"

    # Move compiled binaries to output
    mv ./TinyClaw-${os}* "${OUTPUT_DIR}/${bot_binary}"
    mv ./TinyClawAdmin-${os}* "${OUTPUT_DIR}/${admin_binary}"

    # Copy config files
    mkdir -p "${OUTPUT_DIR}/conf/"
    cp -r ./conf/i18n "${OUTPUT_DIR}/conf/"
    cp -r ./conf/mcp "${OUTPUT_DIR}/conf/"
    cp -r ./conf/img "${OUTPUT_DIR}/conf/"
    mkdir -p "${OUTPUT_DIR}/data/"

    # Copy admin UI files
    mkdir -p "${OUTPUT_DIR}/adminui/"
    cp -r ./admin/adminui/* "${OUTPUT_DIR}/adminui/"

    # Package everything into a tarball
    tar zcf "${RELEASE_DIR}/${release_name}" -C "${OUTPUT_DIR}" .

    # Clean up intermediate files
    rm -rf "${OUTPUT_DIR}"/* ./github.com/*
}

# Platforms to compile (uncomment Windows if needed)
compile_and_package linux amd64
compile_and_package windows amd64
#compile_and_package darwin amd64
#compile_and_package darwin arm64

# Final cleanup
rm -rf "${OUTPUT_DIR}"
echo "✅ Compilation and packaging complete. Output is in ${RELEASE_DIR}"
