#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_ROOT}"

crlf_hits=()
while IFS= read -r path; do
  case "${path}" in
    *.bat|*.cmd|*.ps1)
      continue
      ;;
  esac

  if LC_ALL=C grep -q $'\r' "${path}"; then
    crlf_hits+=("${path}")
  fi
done < <(git grep -I -l '' --)

if ((${#crlf_hits[@]} > 0)); then
  printf 'CRLF line endings detected in tracked text files:\n' >&2
  printf '  %s\n' "${crlf_hits[@]}" >&2
  exit 1
fi

non_exec_scripts=()
while IFS= read -r path; do
  if [[ ! -x "${path}" ]]; then
    non_exec_scripts+=("${path}")
  fi
done < <(git ls-files 'admin/shell/*.sh' 'deploy/docker/*.sh' 'scripts/*.sh')

if ((${#non_exec_scripts[@]} > 0)); then
  printf 'Executable bit is missing for required shell scripts:\n' >&2
  printf '  %s\n' "${non_exec_scripts[@]}" >&2
  exit 1
fi

printf 'Repository hygiene checks passed.\n'
