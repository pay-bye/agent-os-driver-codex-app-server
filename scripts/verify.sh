#!/usr/bin/env bash
set -euo pipefail

main() {
  cd "$(root)"

  case "${1:-}" in
    --unit)
      bash scripts/verify/go.sh
      ;;
    --daemon-readiness)
      bash scripts/verify/integration.sh
      ;;
    "")
      echo "missing mode: expected --unit or --daemon-readiness" >&2
      return 2
      ;;
    *)
      echo "unknown mode: $1" >&2
      return 2
      ;;
  esac
}

root() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  cd "$script_dir/.." && pwd
}

main "$@"
