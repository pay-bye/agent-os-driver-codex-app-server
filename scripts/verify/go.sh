#!/usr/bin/env bash
set -euo pipefail

readonly GO_VERSION="go1.26.4"
readonly UNIT_LIMIT_SECONDS=120
readonly STEPDOWN_PACKAGE="stepdown.dev/go/cmd/stepdown@v0.1.1"
readonly EXPECTED_STEPDOWN_PACKAGE="stepdown.dev/go/cmd/stepdown@v0.1.1"

main() {
  cd "$(root)"
  prepare_go_toolchain
  verify_unit
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

  base_go="$(command -v go || true)"
  if [[ -n "$base_go" ]]; then
    gopath_bin="$("$base_go" env GOPATH)/bin"
    export PATH="$gopath_bin:$PATH"
  fi

  if ! command -v "$GO_VERSION" >/dev/null 2>&1; then
    echo "$GO_VERSION is required. Install with: GOTOOLCHAIN=local go install golang.org/dl/$GO_VERSION@latest && $GO_VERSION download" >&2
    return 1
  fi

  goroot="$("$GO_VERSION" env GOROOT)"
  if [[ ! -x "$goroot/bin/go" || ! -x "$goroot/bin/gofmt" ]]; then
    echo "$GO_VERSION is installed but not downloaded. Run: $GO_VERSION download" >&2
    return 1
  fi

  shim_dir="$(mktemp -d)"
  ln -s "$goroot/bin/go" "$shim_dir/go"
  ln -s "$goroot/bin/gofmt" "$shim_dir/gofmt"
  export PATH="$shim_dir:$PATH"
  export GOTOOLCHAIN=local
  TOOLCHAIN_SHIM_DIR="$shim_dir"
  trap 'rm -rf "${TOOLCHAIN_SHIM_DIR:-}"' EXIT

  verify_toolchain
}

verify_unit() {
  local started
  started="$(now)"

  run_step "$started" "toolchain version verification" 5 verify_toolchain
  run_step "$started" "gofmt clean check" 5 verify_format
  run_step "$started" "architecture tests" 30 bash scripts/verify/architecture.sh
  run_step "$started" "unit tests" 60 go test -count=1 ./...
  run_step "$started" "stepdown" 10 verify_stepdown
  run_step "$started" "schema command verification" 30 verify_schema_command

  echo "unit gate total: $(( "$(now)" - started ))s"
}

run_step() {
  local aggregate_started="$1"
  local label="$2"
  local limit="$3"
  shift 3

  local started elapsed
  started="$(now)"
  "$@"
  elapsed="$(( "$(now)" - started ))"
  echo "$label: ${elapsed}s"
  if (( elapsed > limit )); then
    echo "step exceeded: $label elapsed=${elapsed}s ceiling=${limit}s" >&2
    return 1
  fi
  verify_unit_budget "$aggregate_started"
}

verify_unit_budget() {
  local started="$1"
  local elapsed
  elapsed="$(( "$(now)" - started ))"
  if (( elapsed >= UNIT_LIMIT_SECONDS )); then
    echo "unit gate exceeded: elapsed=${elapsed}s ceiling=${UNIT_LIMIT_SECONDS}s" >&2
    return 1
  fi
}

now() {
  date +%s
}

verify_toolchain() {
  local version
  version="$(go version)"
  if [[ "$version" != go\ version\ "$GO_VERSION"* ]]; then
    echo "go version mismatch: got '$version' want '$GO_VERSION'" >&2
    return 1
  fi
}

verify_stepdown() {
  verify_stepdown_pin
  go run "$STEPDOWN_PACKAGE" ./...
}

verify_stepdown_pin() {
  if [[ "$STEPDOWN_PACKAGE" != "$EXPECTED_STEPDOWN_PACKAGE" ]]; then
    echo "stepdown version mismatch: got '$STEPDOWN_PACKAGE' want '$EXPECTED_STEPDOWN_PACKAGE'" >&2
    return 1
  fi
}

verify_format() {
  local files
  local changed

  mapfile -d '' files < <(find . -name '*.go' -print0)
  if (( ${#files[@]} == 0 )); then
    return 0
  fi

  changed="$(gofmt -l "${files[@]}")"
  if [[ -n "$changed" ]]; then
    echo "gofmt changed files:" >&2
    echo "$changed" >&2
    return 1
  fi
}

verify_schema_command() {
  local output
  output="$(mktemp -d)"
  codex app-server generate-json-schema --experimental --out "$output" >/dev/null

  test -f "$output/codex_app_server_protocol.v2.schemas.json"
  test -f "$output/v2/ThreadStartParams.json"
  test -f "$output/v2/TurnStartParams.json"
  test -f "$output/v2/TurnCompletedNotification.json"
}

main "$@"
