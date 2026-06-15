#!/usr/bin/env bash
set -euo pipefail

main() {
  cd "$(root)"
  prepare_go_toolchain
  verify_daemon
}

root() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  cd "$script_dir/../.." && pwd
}

prepare_go_toolchain() {
  local base_go
  local gopath_bin
  local goroot
  local shim_dir
  local go_version

  go_version="go1.26.4"
  base_go="$(command -v go || true)"
  if [[ -n "$base_go" ]]; then
    gopath_bin="$("$base_go" env GOPATH)/bin"
    export PATH="$gopath_bin:$PATH"
  fi

  if ! command -v "$go_version" >/dev/null 2>&1; then
    echo "$go_version is required. Install with: GOTOOLCHAIN=local go install golang.org/dl/$go_version@latest && $go_version download" >&2
    return 1
  fi

  goroot="$("$go_version" env GOROOT)"
  if [[ ! -x "$goroot/bin/go" || ! -x "$goroot/bin/gofmt" ]]; then
    echo "$go_version is installed but not downloaded. Run: $go_version download" >&2
    return 1
  fi

  shim_dir="$(mktemp -d)"
  ln -s "$goroot/bin/go" "$shim_dir/go"
  ln -s "$goroot/bin/gofmt" "$shim_dir/gofmt"
  export PATH="$shim_dir:$PATH"
  export GOTOOLCHAIN=local
  TOOLCHAIN_SHIM_DIR="$shim_dir"
  trap 'rm -rf "${TOOLCHAIN_SHIM_DIR:-}"' EXIT
}

verify_daemon() {
  local started
  started="$(now)"

  RUN_DAEMON_READINESS=1 go test -count=1 ./tests/integration/daemon -run TestLiveDaemonReadiness -v
  echo "isolated daemon readiness: $(( "$(now)" - started ))s"
  echo "daemon readiness total: $(( "$(now)" - started ))s"
}

now() {
  date +%s
}

main "$@"
