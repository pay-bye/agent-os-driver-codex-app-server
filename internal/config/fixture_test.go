package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validConfig() Config {
	return Config{
		InvocationBaseURL: "http://127.0.0.1:8080",
		ChannelKey:        "q01",
		LeaseSeconds:      60,
		CodexBin:          "/usr/bin/codex",
		CodexHome:         "/tmp/codex-home",
		ControlEndpoint:   "unix:///tmp/codex-home/codex-app-server.sock",
		WorkspaceRoot:     "/tmp/work",
		InputTextPointer:  "/work/prompt",
		CompletionNeeds:   []Need{{Kind: "done"}},
		FailureNeeds:      []Need{{Kind: "failed"}},
		RedactionMode:     "metadata_only",
	}
}

func writeConfig(t *testing.T, item Config) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	content, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func acceptCodex(string) error {
	return nil
}

func requireError(t *testing.T, err error, expected string) {
	t.Helper()

	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error containing %q, got %v", expected, err)
	}
}
