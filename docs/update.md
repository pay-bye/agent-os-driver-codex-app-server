# Update Codex App-Server Driver

Updates use tagged public module releases from
`github.com/pay-bye/agent-os-driver-codex-app-server`.

Public update commands become adopter-truth only after U3 rewrites public module paths, U6
publishes public release artifacts, and U8 accepts clean-machine proof.

```sh
go install github.com/pay-bye/agent-os-driver-codex-app-server/cmd/driver@v0.1.0-rc.1
driver doctor "$HOME/.agent-os/drivers/codex-app-server"
```

Before upgrading, verify the published invocation contract, the driver manifest, and release
metadata. The catalog links to those sources; it does not decide whether the update is compatible.
