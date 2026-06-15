# Governance

Codex app-server driver maintainers own the driver binary, driver compatibility manifest, release
metadata, and adopter documentation in `github.com/pay-bye/agent-os-driver-codex-app-server`.

## Decision Ownership

- Invocation behavior follows the published Agent OS invocation contract.
- Codex app-server compatibility is recorded in `internal/install/driver_manifest.json`.
- Public release compatibility is governed by this repository's release metadata.

## Catalog Boundary

The future `github.com/pay-bye/agent-os-catalog` repository is discovery only. It links to the
driver manifest, release metadata, and published invocation contract. Catalog entries do not decide
compatibility.

## Release Authority

Public release authority begins when U3 rewrites public module paths, U6 publishes release
artifacts, and U8 accepts clean-machine proof.
