package install

import (
	"context"
	"errors"
	"testing"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
)

func TestInstallerWritesOwnedFilesAfterCompatibilityCheck(t *testing.T) {
	verifier := &fakeVerifier{result: resultFixture()}
	path := writeConfig(t, "http://127.0.0.1:8080")
	home := t.TempDir()

	record, err := Installer{CodexCheck: acceptCodex, Verifier: verifier}.Apply(context.Background(), path, home)
	if err != nil {
		t.Fatal(err)
	}

	if verifier.calls != 1 {
		t.Fatalf("compatibility checks = %d, want 1", verifier.calls)
	}
	if record.LastDiagnosticCode != "compatible" {
		t.Fatalf("diagnostic = %q, want compatible", record.LastDiagnosticCode)
	}
	requireFiles(t, home, []string{
		"install.json",
		"protocol/required-schemas.json",
		"runner/registration.json",
	})
}

func TestInstallerWritesNoRecordWhenCompatibilityFails(t *testing.T) {
	verifier := &fakeVerifier{err: compatibility.ErrFeatureMissing}
	home := t.TempDir()

	_, err := Installer{
		CodexCheck: acceptCodex,
		Verifier:   verifier,
	}.Apply(context.Background(), writeConfig(t, "http://127.0.0.1:8080"), home)

	if !errors.Is(err, compatibility.ErrFeatureMissing) {
		t.Fatalf("error = %v, want feature_missing", err)
	}
	requireMissing(t, home, "install.json")
}

func TestApplyRejectsDifferentExistingInstall(t *testing.T) {
	home := t.TempDir()
	if _, err := applyInstalled(t, writeConfig(t, "http://127.0.0.1:8080"), home); err != nil {
		t.Fatal(err)
	}

	_, err := applyInstalled(t, writeConfig(t, "http://127.0.0.1:8081"), home)

	requireError(t, err, "install_conflict")
}
