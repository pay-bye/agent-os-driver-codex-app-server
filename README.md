# Codex App-Server Driver

The Codex app-server driver installs, runs, and removes the selected Codex app-server invocation
driver for Agent OS.

## Public Coordinates

Future public repository: `github.com/pay-bye/agent-os-driver-codex-app-server`

Future public module: `github.com/pay-bye/agent-os-driver-codex-app-server`

Public module commands become adopter-truth only after U3 rewrites public module paths, U6
publishes public release artifacts, and U8 accepts clean-machine proof.

## Documents

- [Install](docs/install.md)
- [Update](docs/update.md)
- [Governance](docs/governance.md)
- [Catalog Cross-Reference](docs/catalog.md)
- [Contributing](CONTRIBUTING.md)
- [Security](SECURITY.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)

## Component-Owned Truth

Driver compatibility evidence is owned by this repository's driver manifest and release metadata:

- `internal/install/driver_manifest.json`
- GitHub release artifacts and attestations published from this repository

The future catalog points to these sources. It does not decide compatibility.

## Commands

- `driver install <config.json> <home>` writes driver-owned configuration under `<home>`.
- `driver run <home>` claims work through the invocation HTTP routes and starts Codex turns through an initialized WebSocket-over-UDS app-server connection.
- `driver remove <home>` removes driver-owned files and runner registration.
- `driver status <home>` prints metadata counters only.
- `driver doctor <home>` reruns compatibility checks without mutating the install record.

## Codex Gate

This driver records the current Codex CLI and requires the app-server protocol subset used by the selected driver:

- Schema canonicalization: `json_sort_keys_v1`
- App-server schema digest: recorded as observed compatibility evidence

Install or update Codex with:

```sh
curl -fsSL https://chatgpt.com/codex/install.sh | sh
codex --version
```

If the current Codex version changes, rerun `driver doctor` and the daemon readiness gate. A version string change is recorded as evidence; protocol-subset, isolation, or cleanup failure remains blocking.

## Configuration

Configuration is JSON. Public examples use opaque ids and localhost endpoints:

```json
{
  "invocation_base_url": "http://127.0.0.1:8080",
  "channel_key": "q01",
  "lease_seconds": 60,
  "codex_bin": "/usr/local/bin/codex",
  "codex_home": "/tmp/codex-home",
  "control_endpoint": "unix:///tmp/codex-home/codex-app-server.sock",
  "workspace_root": "/tmp/work",
  "input_text_pointer": "/prompt",
  "completion_needs": [],
  "failure_needs": [],
  "redaction_mode": "metadata_only"
}
```

The driver records ids, counts, and error codes. It does not print work payloads, prompt text, or credential values.

The control endpoint must be a `unix://` socket inside `codex_home`. Driver tests use temporary homes and test-owned
endpoints; they do not use the operator's live Codex app-server or real `CODEX_HOME`.

## Verification

Run the default daemon-free gate:

```sh
./scripts/verify.sh --unit
```

Run the non-default isolated daemon readiness gate only on a machine with the supported Codex CLI:

```sh
./scripts/verify.sh --daemon-readiness
```

The readiness gate creates process-local home, config, cache, runtime, temp, and workspace paths under one temporary root. It seeds the managed standalone path from the verified Codex binary, starts the test-owned daemon, runs the selected WebSocket-over-UDS control client through `thread/start`, stops the daemon, and fails instead of silently skipping when prerequisites are absent.
