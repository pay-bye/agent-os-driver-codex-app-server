# Install Codex App-Server Driver

The driver is installed from `github.com/pay-bye/agent-os-driver-codex-app-server`.

Public install commands become adopter-truth only after U3 rewrites public module paths, U6
publishes public release artifacts, and U8 accepts clean-machine proof. Until those gates pass,
this command documents the accepted public coordinate shape only.

```sh
go install github.com/pay-bye/agent-os-driver-codex-app-server/cmd/driver@v0.1.0-rc.1
driver install config.json "$HOME/.agent-os/drivers/codex-app-server"
driver doctor "$HOME/.agent-os/drivers/codex-app-server"
```

Configuration format is described in the root README. Compatibility is determined by the driver
manifest, published invocation contract, and release metadata. The catalog links to those sources;
it does not decide compatibility.
