package config

import (
	"errors"
	"testing"
)

func TestReadRejectsDatabaseEndpoint(t *testing.T) {
	item := validConfig()
	item.InvocationBaseURL = "postgres://127.0.0.1/db"

	_, err := Read(writeConfig(t, item), acceptCodex)

	requireError(t, err, "invalid_invocation_base_url")
}

func TestReadRejectsMissingCodexHome(t *testing.T) {
	item := validConfig()
	item.CodexHome = ""

	_, err := Read(writeConfig(t, item), acceptCodex)

	requireError(t, err, "invalid_codex_home")
}

func TestReadRejectsUnavailableCodexCommand(t *testing.T) {
	_, err := Read(writeConfig(t, validConfig()), func(string) error {
		return errors.New("missing")
	})

	requireError(t, err, "codex_unavailable")
}
