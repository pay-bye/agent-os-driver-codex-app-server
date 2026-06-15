#!/usr/bin/env bash
set -euo pipefail

root() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  cd "$script_dir/../.." && pwd
}

cd "$(root)"
go test -count=1 ./tests/architecture
