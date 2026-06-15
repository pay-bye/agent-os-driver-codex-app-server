package install

import (
	_ "embed"
	"path/filepath"
)

func writeOwnedFiles(home string) error {
	if err := writeManifestFiles(home); err != nil {
		return err
	}
	return writeConfigFiles(home)
}

func writeManifestFiles(home string) error {
	return writeFile(filepath.Join(home, "protocol", "required-schemas.json"), []byte(requiredSchemaContent()))
}

func requiredSchemaContent() string {
	return `{
  "required": [
    "codex_app_server_protocol.schemas.json",
    "codex_app_server_protocol.v2.schemas.json",
    "v2/ThreadStartParams.json",
    "v2/ThreadStartResponse.json",
    "v2/TurnStartParams.json",
    "v2/TurnStartResponse.json",
    "v2/TurnCompletedNotification.json",
    "v2/TurnInterruptParams.json",
    "v2/TurnInterruptResponse.json",
    "v2/RemoteControlEnableResponse.json",
    "v2/RemoteControlDisableResponse.json",
    "v2/RemoteControlStatusReadResponse.json",
    "v2/RemoteControlStatusChangedNotification.json"
  ]
}
`
}
