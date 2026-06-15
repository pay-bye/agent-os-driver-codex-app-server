package main

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
)

func TestStatusReadsLocalRecordWithoutEndpointCalls(t *testing.T) {
	server := newRouteServer(t)
	defer server.Close()
	home := writeRecord(t, server.URL)

	if err := statusCommand([]string{home}, commandProcess(io.Discard)); err != nil {
		t.Fatal(err)
	}

	requireRoutes(t, server.routes, nil)
}

func TestStatusReportsMissingRecordDiagnostic(t *testing.T) {
	var output bytes.Buffer
	err := statusCommand([]string{t.TempDir()}, commandProcess(&output))

	if !errors.Is(err, compatibility.ErrInstallRecordUpgradeRequired) {
		t.Fatalf("error = %v, want install_record_upgrade_required", err)
	}
	requireLocalDiagnostic(t, output.String())
}

func TestStatusReportsMalformedRecordDiagnostic(t *testing.T) {
	home := t.TempDir()
	writeMalformedRecord(t, home)

	var output bytes.Buffer
	err := statusCommand([]string{home}, commandProcess(&output))

	if !errors.Is(err, compatibility.ErrInstallRecordUpgradeRequired) {
		t.Fatalf("error = %v, want install_record_upgrade_required", err)
	}
	requireLocalDiagnostic(t, output.String())
}
