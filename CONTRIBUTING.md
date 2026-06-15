# Contributing

Contributions keep the Codex app-server driver aligned with the published invocation contract and
driver-owned compatibility manifest.

## Before Opening a Pull Request

1. Keep public examples on `github.com/pay-bye/agent-os-driver-codex-app-server`.
2. Do not present a local source checkout, local source substitution, or private source path as an
   adopter install or update path.
3. Keep compatibility facts in `internal/install/driver_manifest.json` and release metadata.
4. Keep catalog references discovery-only.
5. Run the unit gate:

```sh
GOTOOLCHAIN=local bash scripts/verify.sh --unit
```

## Code And Docs Standard

- Changes are scoped to one logical purpose.
- Driver behavior is verified through public invocation routes.
- Public docs state future gates when public artifacts are not yet released.

## License

By contributing, you agree that your contribution is licensed under Apache-2.0.
