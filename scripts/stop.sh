#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

cd "${REPO_ROOT}"

source_env_file "${ENV_FILE}"

if [[ "${1:-}" == "--down" ]]; then
  docker_compose down
  echo "Containers stopped with --down."
  exit 0
fi

echo "stop.sh now keeps containers running."
echo "Use './scripts/stop.sh --down' only when you intentionally want to stop the Compose stack."
echo
docker_compose ps
