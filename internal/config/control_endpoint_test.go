package config

import "testing"

func TestReadRejectsNonLoopbackEndpoint(t *testing.T) {
	item := validConfig()
	item.ControlEndpoint = "ws://10.0.0.5:8080"

	_, err := Read(writeConfig(t, item), acceptCodex)

	requireError(t, err, "invalid_control_endpoint")
}

func TestReadRejectsLoopbackWebsocketEndpoint(t *testing.T) {
	item := validConfig()
	item.ControlEndpoint = "ws://127.0.0.1:8080"

	_, err := Read(writeConfig(t, item), acceptCodex)

	requireError(t, err, "invalid_control_endpoint")
}

func TestReadRejectsControlSocketOutsideCodexHome(t *testing.T) {
	item := validConfig()
	item.ControlEndpoint = "unix:///tmp/other-home/codex-app-server.sock"

	_, err := Read(writeConfig(t, item), acceptCodex)

	requireError(t, err, "invalid_control_endpoint")
}
