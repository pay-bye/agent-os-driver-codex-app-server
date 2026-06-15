package compatibility

import (
	"reflect"
	"testing"
)

func requireEqual[T any](t *testing.T, got T, want T) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func requiredSchemaFiles() []string {
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
