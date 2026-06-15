package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveMissingInstallRecordSucceeds(t *testing.T) {
	home := t.TempDir()

	if err := Remove(home); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveDeletesOwnedFilesAndPreservesCodexMaterial(t *testing.T) {
	home := t.TempDir()
	if _, err := applyInstalled(t, writeConfig(t, "http://127.0.0.1:8080"), home); err != nil {
		t.Fatal(err)
	}
	preserved := filepath.Join(home, "codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(preserved), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(preserved, []byte("local-user-config"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Remove(home); err != nil {
		t.Fatal(err)
	}

	requireMissing(t, home, "install.json")
	requireMissing(t, home, "protocol/required-schemas.json")
	requireMissing(t, home, "runner/registration.json")
	if _, err := os.Stat(preserved); err != nil {
		t.Fatalf("expected preserved Codex material: %v", err)
	}
}
