package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/control"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/install"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/invoke"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/status"
)

type socketStatus interface {
	Ready(context.Context) bool
}

type doctorChecks struct {
	App    compatibility.AppMetadataReader
	Socket socketStatus
	Run    control.CommandRunner
}

type diagnostic struct {
	record       install.Record
	requirements compatibility.Requirements
	checks       doctorChecks
	env          []string
}

type fixedAppMetadata struct {
	metadata compatibility.AppMetadata
}

func (p fixedAppMetadata) Metadata(context.Context) (compatibility.AppMetadata, error) {
	return p.metadata, nil
}

func runDiagnostic(ctx context.Context, current diagnostic) (compatibility.Result, error) {
	metadata, err := appMetadata(ctx, current)
	if err != nil {
		return compatibility.Result{DiagnosticCode: compatibility.Code(err)}, err
	}
	if err := requireSocketReady(ctx, current.record, current.checks.Socket); err != nil {
		return compatibility.Result{DiagnosticCode: compatibility.Code(err)}, err
	}
	if err := control.ReadDaemonVersion(ctx, current.record.Config, current.checks.Run); err != nil {
		return compatibility.Result{DiagnosticCode: compatibility.Code(err)}, err
	}
	return verifyInvocation(ctx, current, metadata)
}

func appMetadata(ctx context.Context, current diagnostic) (compatibility.AppMetadata, error) {
	metadata, err := metadataReader(current).Metadata(ctx)
	if err != nil {
		return compatibility.AppMetadata{}, err
	}
	if err := current.requirements.CheckApp(metadata); err != nil {
		return compatibility.AppMetadata{}, err
	}
	return metadata, nil
}

func metadataReader(current diagnostic) compatibility.AppMetadataReader {
	if current.checks.App != nil {
		return current.checks.App
	}
	return compatibility.CommandAppMetadata{
		CodexBin: current.record.Config.CodexBin,
		Env:      current.env,
	}
}

func verifyInvocation(
	ctx context.Context,
	current diagnostic,
	metadata compatibility.AppMetadata,
) (compatibility.Result, error) {
	result, err := compatibility.Verifier{
		Requirements: current.requirements,
		Invocation:   invoke.New(current.record.Config.InvocationBaseURL, nil),
		App:          fixedAppMetadata{metadata: metadata},
	}.Verify(ctx)
	return result, err
}

func doctor(home string, app compatibility.AppMetadataReader, item process) error {
	return diagnose(home, doctorChecks{App: app}, item)
}

func diagnose(home string, checks doctorChecks, item process) error {
	record, err := loadRecord(home)
	if err != nil {
		fmt.Fprintln(item.stdout, status.RenderDiagnostic("local", compatibility.Code(err)))
		return err
	}
	requirements, err := install.DefaultRequirements()
	if err != nil {
		return err
	}
	if err := requirements.CheckRecordVersion(record.StoredVersion()); err != nil {
		fmt.Fprintln(item.stdout, status.RenderDiagnostic("local", compatibility.Code(err)))
		return err
	}
	if checks.Run == nil {
		checks.Run = control.NewCommandRunner(item.env)
	}
	result, err := runDiagnostic(context.Background(), diagnostic{
		record:       record,
		requirements: requirements,
		checks:       checks,
		env:          item.env,
	})
	fmt.Fprintln(item.stdout, status.RenderDiagnostic("live", result.DiagnosticCode))
	return err
}

func requireSocketReady(ctx context.Context, record install.Record, socket socketStatus) error {
	if socket == nil {
		socket = control.NewUnixClient(record.Config.ControlEndpoint)
	}
	if socket.Ready(ctx) {
		return nil
	}
	return errors.New("control_unavailable")
}
