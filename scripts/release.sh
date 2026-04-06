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

# Build the admin binary (locally for the specified platform)
build_admin_local() {
    local os=$1
    local arch=$2
    local ext=""
    [[ "$os" == "windows" ]] && ext=".exe"

    local output_name="TinyClawAdmin"
    echo "=============================="
    echo "Building admin [$os/$arch] using go build..."
    echo "=============================="

    GOOS=$os GOARCH=$arch CGO_ENABLED=1 go build -o "${OUTPUT_DIR}/${output_name}" ./admin
}

# Build main binary + package everything
compile_and_package_local() {
    local os=$1
    local arch=$2
    local ext=""
    [[ "$os" == "windows" ]] && ext=".exe"

    echo "=============================="
    echo "Building TinyClaw [$os/$arch] using go build..."
    echo "=============================="

    local bot_output="TinyClaw"

    # Build main bot binary
    GOOS=$os GOARCH=$arch CGO_ENABLED=1 go build -o "${OUTPUT_DIR}/${bot_output}" ./cmd/tinyclaw

    # Build admin binary
    build_admin_local $os $arch

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
    local release_name="TinyClaw-${os}-${arch}.tar.gz"
    tar zcf "${RELEASE_DIR}/${release_name}" -C "${OUTPUT_DIR}" .

    echo "✅ Packaged ${release_name}"
}

# Platforms to compile
#compile_and_package linux amd64
#compile_and_package windows amd64
compile_and_package_local darwin amd64
compile_and_package_local darwin arm64

# Final cleanup
rm -rf "${OUTPUT_DIR}"
echo "✅ Compilation and packaging complete. Output is in ${RELEASE_DIR}"
