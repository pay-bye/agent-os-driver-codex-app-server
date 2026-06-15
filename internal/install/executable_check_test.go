package install

import (
	"context"
	"testing"
)

func TestInstallerChecksExecutableWhenNoOverrideIsConfigured(t *testing.T) {
	verifier := &fakeVerifier{result: resultFixture()}
	home := t.TempDir()

	_, err := Installer{Verifier: verifier}.Apply(context.Background(), writeConfig(t, "http://127.0.0.1:8080"), home)

	requireError(t, err, "codex_unavailable")
	if verifier.calls != 0 {
		t.Fatalf("compatibility checks = %d, want 0", verifier.calls)
	}
	requireMissing(t, home, "install.json")
}
