package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
)

func TestRunRejectsMissingRecordBeforeWorkRoutes(t *testing.T) {
	err := runCommand([]string{t.TempDir()}, commandProcess(io.Discard))

	if !errors.Is(err, compatibility.ErrInstallRecordUpgradeRequired) {
		t.Fatalf("error = %v, want install_record_upgrade_required", err)
	}
}

func TestRunRejectsMalformedRecordBeforeWorkRoutes(t *testing.T) {
	home := t.TempDir()
	writeMalformedRecord(t, home)

	err := runCommand([]string{home}, commandProcess(io.Discard))

	if !errors.Is(err, compatibility.ErrInstallRecordUpgradeRequired) {
		t.Fatalf("error = %v, want install_record_upgrade_required", err)
	}
}

func TestRemoveDeletesUnreadableRecordWithoutEndpointCalls(t *testing.T) {
	server := newRouteServer(t)
	defer server.Close()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "install.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := removeCommand([]string{home}); err != nil {
		t.Fatal(err)
	}

	requireRoutes(t, server.routes, nil)
	if _, err := os.Stat(filepath.Join(home, "install.json")); err == nil {
		t.Fatal("install record still exists")
	}
}
