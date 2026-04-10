#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

SCRIPT_HINT_CMD="./scripts/bootstrap-host.sh"

script_section "Checking Docker"
if docker ps >/dev/null 2>&1; then
  script_ok "Docker daemon is reachable from WSL"
elif [[ -x "${WINDOWS_DOCKER}" ]] && "${WINDOWS_DOCKER}" version >/dev/null 2>&1; then
  script_ok "Docker Desktop is reachable through the Windows backend"
else
  script_error "Docker daemon is unavailable. Start Docker Desktop first, then rerun this script."
  exit 1
fi

script_section "Checking Go"
if command -v go >/dev/null 2>&1 && go version | grep -q 'go1\.24'; then
  script_ok "Go is already installed"
  script_kv "Version" "$(go version)"
  exit 0
fi

if ! command -v python3 >/dev/null 2>&1; then
  script_error "python3 is required to resolve the latest Go 1.24.x version."
  exit 1
fi

script_info "Resolving the latest Go 1.24.x release"
GO_VERSION="$(
  curl -fsSL 'https://go.dev/dl/?mode=json&include=all' | python3 -c '
import json, sys
data = json.load(sys.stdin)
versions = []
for item in data:
    version = item.get("version", "")
    if not version.startswith("go1.24."):
        continue
    parts = tuple(int(x) for x in version[4:].split("."))
    versions.append((parts, version))
if not versions:
    raise SystemExit("no Go 1.24.x release found")
versions.sort(reverse=True)
print(versions[0][1])
'
)"
script_ok "Resolved Go release"
script_kv "Version" "${GO_VERSION}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

GO_ARCHIVE="${GO_VERSION}.linux-amd64.tar.gz"
GO_INSTALL_ROOT="/usr/local"
GO_BIN_PATH="/usr/local/go/bin"

script_section "Installing Go"
script_info "Downloading ${GO_ARCHIVE}"
curl -fsSL "https://go.dev/dl/${GO_ARCHIVE}" -o "${TMP_DIR}/${GO_ARCHIVE}"

if sudo -n true >/dev/null 2>&1; then
  script_info "Installing to ${GO_INSTALL_ROOT}"
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf "${TMP_DIR}/${GO_ARCHIVE}"
else
  GO_INSTALL_ROOT="${HOME}/.local"
  GO_BIN_PATH="${HOME}/.local/go/bin"
  script_warn "sudo is unavailable; installing to ${GO_INSTALL_ROOT} instead"
  rm -rf "${HOME}/.local/go"
  mkdir -p "${HOME}/.local"
  tar -C "${HOME}/.local" -xzf "${TMP_DIR}/${GO_ARCHIVE}"
fi

if ! grep -q "${GO_BIN_PATH}" "${HOME}/.profile" 2>/dev/null; then
  printf '\nexport PATH=%s:$PATH\n' "${GO_BIN_PATH}" >> "${HOME}/.profile"
  script_info "Added ${GO_BIN_PATH} to ~/.profile"
fi

export PATH="${GO_BIN_PATH}:${PATH}"

script_ok "Go installation completed"
script_kv "Install root" "${GO_INSTALL_ROOT}"
script_kv "Go bin path" "${GO_BIN_PATH}"
script_kv "go version" "$(go version)"
