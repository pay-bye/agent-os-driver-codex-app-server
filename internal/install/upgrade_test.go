package install

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
)

func TestInstallerUpgradeRewritesReadableRecord(t *testing.T) {
	home := t.TempDir()
	if _, err := applyInstalled(t, writeConfig(t, "http://127.0.0.1:8080"), home); err != nil {
		t.Fatal(err)
	}
	writeLegacyVersion(t, home)

	record, err := Installer{
		CodexCheck: acceptCodex,
		Verifier:   &fakeVerifier{result: resultFixture()},
	}.Upgrade(context.Background(), home)

	if err != nil {
		t.Fatal(err)
	}
	if record.StoredVersion() != 3 {
		t.Fatalf("stored version = %d, want 3", record.StoredVersion())
	}
}

func TestInstallerUpgradeRejectsLegacyRecordWithoutCodexHome(t *testing.T) {
	home := t.TempDir()
	if _, err := applyInstalled(t, writeConfig(t, "http://127.0.0.1:8080"), home); err != nil {
		t.Fatal(err)
	}
	writeLegacyRecordWithoutCodexHome(t, home)
	before := readInstallRecord(t, home)
	verifier := &fakeVerifier{result: resultFixture()}

	_, err := Installer{
		CodexCheck: acceptCodex,
		Verifier:   verifier,
	}.Upgrade(context.Background(), home)

	if !errors.Is(err, compatibility.ErrInstallRecordUpgradeRequired) {
		t.Fatalf("error = %v, want install_record_upgrade_required", err)
	}
	if verifier.calls != 0 {
		t.Fatalf("compatibility checks = %d, want 0", verifier.calls)
	}
	after := readInstallRecord(t, home)
	if string(after) != string(before) {
		t.Fatal("upgrade rewrote record missing codex_home")
	}
}

func TestInstallerUpgradePreservesRecordWhenCompatibilityFails(t *testing.T) {
	home := t.TempDir()
	if _, err := applyInstalled(t, writeConfig(t, "http://127.0.0.1:8080"), home); err != nil {
		t.Fatal(err)
	}
	before := readInstallRecord(t, home)

	_, err := Installer{
		CodexCheck: acceptCodex,
		Verifier:   &fakeVerifier{err: compatibility.ErrSchemaDigestUnaccepted},
	}.Upgrade(context.Background(), home)

	if !errors.Is(err, compatibility.ErrSchemaDigestUnaccepted) {
		t.Fatalf("error = %v, want schema_digest_unaccepted", err)
	}
	after := readInstallRecord(t, home)
	if string(after) != string(before) {
		t.Fatal("upgrade changed record after compatibility failure")
	}
}

func TestInstallerUpgradePreservesMalformedRecordWithDiagnostic(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(recordPath(home), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := readInstallRecord(t, home)
	verifier := &fakeVerifier{result: resultFixture()}

	_, err := Installer{
		CodexCheck: acceptCodex,
		Verifier:   verifier,
	}.Upgrade(context.Background(), home)

	if !errors.Is(err, compatibility.ErrInstallRecordUpgradeRequired) {
		t.Fatalf("error = %v, want install_record_upgrade_required", err)
	}
	if verifier.calls != 0 {
		t.Fatalf("compatibility checks = %d, want 0", verifier.calls)
	}
	after := readInstallRecord(t, home)
	if string(after) != string(before) {
		t.Fatal("upgrade changed malformed record")
	}
}

func TestInstallerUpgradeReportsMissingRecordDiagnostic(t *testing.T) {
	verifier := &fakeVerifier{result: resultFixture()}

	_, err := Installer{
		CodexCheck: acceptCodex,
		Verifier:   verifier,
	}.Upgrade(context.Background(), t.TempDir())

	if !errors.Is(err, compatibility.ErrInstallRecordUpgradeRequired) {
		t.Fatalf("error = %v, want install_record_upgrade_required", err)
	}
	if verifier.calls != 0 {
		t.Fatalf("compatibility checks = %d, want 0", verifier.calls)
	}
}

func writeLegacyVersion(t *testing.T, home string) {
	t.Helper()

	record, err := Load(home)
	if err != nil {
		t.Fatal(err)
	}
	record.Version = 0
	content, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(recordPath(home), content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeLegacyRecordWithoutCodexHome(t *testing.T, home string) {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal(readInstallRecord(t, home), &record); err != nil {
		t.Fatal(err)
	}
	record["install_record_version"] = float64(1)
	item, ok := record["config"].(map[string]any)
	if !ok {
		t.Fatal("install record config is not an object")
	}
	delete(item, "codex_home")
	item["control_endpoint"] = "unix:///tmp/codex-app-server.sock"
	writeRecordMap(t, home, record)
}

func writeRecordMap(t *testing.T, home string, record map[string]any) {
	t.Helper()

	content, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(recordPath(home), content, 0o644); err != nil {
		t.Fatal(err)
	}
}
