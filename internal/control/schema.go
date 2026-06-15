package control

import (
	"fmt"
	"os"
	"path/filepath"
)

func RequireSchemas(root string) error {
	for _, path := range requiredSchemas() {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			return fmt.Errorf("schema_missing: %s", path)
		}
	}
	return nil
}

func requiredSchemas() []string {
	return []string{
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
		"v2/RemoteControlStatusChangedNotification.json",
	}
}
