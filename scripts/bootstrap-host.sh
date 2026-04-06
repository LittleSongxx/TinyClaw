#!/usr/bin/env bash
set -euo pipefail

WINDOWS_DOCKER="/mnt/c/Program Files/Docker/Docker/resources/bin/docker.exe"

if docker ps >/dev/null 2>&1; then
  :
elif [[ -x "${WINDOWS_DOCKER}" ]] && "${WINDOWS_DOCKER}" version >/dev/null 2>&1; then
  :
else
  echo "Docker daemon is unavailable. Start Docker Desktop first, then rerun this script." >&2
  exit 1
fi

if command -v go >/dev/null 2>&1 && go version | grep -q 'go1\.24'; then
  echo "Go already installed: $(go version)"
  exit 0
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required to resolve the latest Go 1.24.x version." >&2
  exit 1
fi

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

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

GO_ARCHIVE="${GO_VERSION}.linux-amd64.tar.gz"
curl -fsSL "https://go.dev/dl/${GO_ARCHIVE}" -o "${TMP_DIR}/${GO_ARCHIVE}"
GO_INSTALL_ROOT="/usr/local"
GO_BIN_PATH="/usr/local/go/bin"

if sudo -n true >/dev/null 2>&1; then
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf "${TMP_DIR}/${GO_ARCHIVE}"
else
  GO_INSTALL_ROOT="${HOME}/.local"
  GO_BIN_PATH="${HOME}/.local/go/bin"
  rm -rf "${HOME}/.local/go"
  mkdir -p "${HOME}/.local"
  tar -C "${HOME}/.local" -xzf "${TMP_DIR}/${GO_ARCHIVE}"
fi

if ! grep -q "${GO_BIN_PATH}" "${HOME}/.profile" 2>/dev/null; then
  printf '\nexport PATH=%s:$PATH\n' "${GO_BIN_PATH}" >> "${HOME}/.profile"
fi

export PATH="${GO_BIN_PATH}:${PATH}"
go version
