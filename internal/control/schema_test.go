package control

import "testing"

func TestRequireSchemasRejectsMissingTurnStart(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "codex_app_server_protocol.v2.schemas.json", "{}")
	writeFile(t, root, "v2/ThreadStartParams.json", "{}")
	writeFile(t, root, "v2/ThreadStartResponse.json", "{}")
	writeFile(t, root, "v2/TurnStartResponse.json", "{}")
	writeFile(t, root, "v2/TurnCompletedNotification.json", "{}")

	err := RequireSchemas(root)

	requireError(t, err, "schema_missing")
}
