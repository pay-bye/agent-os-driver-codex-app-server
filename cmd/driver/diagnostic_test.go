package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/control"
)

func TestDoctorCallsOnlyReadOnlyCompatibilityEndpoints(t *testing.T) {
	server := newRouteServer(t)
	defer server.Close()
	home := writeRecord(t, server.URL)
	before := readRecord(t, home)
	socket := &readySocket{}
	commands := &commandLog{}

	err := diagnose(home, doctorChecks{
		App: appReader{metadata: compatibility.AppMetadata{
			CodexVersion:   currentVersion,
			SchemaDigest:   appDigest,
			SchemaFiles:    requiredSchemaFiles(),
			Methods:        []string{"initialize", "thread/start", "turn/start", "turn/interrupt"},
			Notifications:  []string{"turn/completed"},
			ControlSurface: "uds_websocket",
		}},
		Socket: socket,
		Run:    commands.Run,
	}, commandProcess(io.Discard))

	if err != nil {
		t.Fatal(err)
	}
	requireRoutes(t, server.routes, []string{"/compatibility"})
	if socket.calls != 1 {
		t.Fatalf("socket checks = %d, want 1", socket.calls)
	}
	requireCommands(t, commands.Commands(), []string{"/usr/bin/codex app-server daemon version"})
	after := readRecord(t, home)
	if string(after) != string(before) {
		t.Fatalf("doctor mutated install record")
	}
}

func TestDoctorReportsMissingRecordBeforeLiveChecks(t *testing.T) {
	var output bytes.Buffer
	err := doctor(
		t.TempDir(),
		appReader{err: errors.New("app reader called")},
		commandProcess(&output),
	)

	if !errors.Is(err, compatibility.ErrInstallRecordUpgradeRequired) {
		t.Fatalf("error = %v, want install_record_upgrade_required", err)
	}
	requireLocalDiagnostic(t, output.String())
}

func TestDoctorReportsMalformedRecordBeforeLiveChecks(t *testing.T) {
	home := t.TempDir()
	writeMalformedRecord(t, home)

	var output bytes.Buffer
	err := doctor(home, appReader{err: errors.New("app reader called")}, commandProcess(&output))

	if !errors.Is(err, compatibility.ErrInstallRecordUpgradeRequired) {
		t.Fatalf("error = %v, want install_record_upgrade_required", err)
	}
	requireLocalDiagnostic(t, output.String())
}

type appReader struct {
	metadata compatibility.AppMetadata
	err      error
}

func (p appReader) Metadata(context.Context) (compatibility.AppMetadata, error) {
	return p.metadata, p.err
}

type readySocket struct {
	calls int
}

func (p *readySocket) Ready(context.Context) bool {
	p.calls++
	return true
}

type commandLog struct {
	commands []string
}

func (l *commandLog) Run(_ context.Context, item control.Command) error {
	l.commands = append(l.commands, strings.Join(append([]string{item.Name}, item.Args...), " "))
	return nil
}

func (l *commandLog) Commands() []string {
	return append([]string(nil), l.commands...)
}
