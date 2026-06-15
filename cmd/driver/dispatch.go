package main

import (
	"context"
	"fmt"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/control"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/install"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/invoke"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/status"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/transmit"
	"os"
	"os/signal"
	"time"
)

func installCommand(args []string, item process) error {
	if len(args) != 2 {
		return usage()
	}
	_, err := install.Installer{Env: item.env}.Apply(context.Background(), args[0], args[1])
	return err
}

func removeCommand(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	return install.Remove(args[0])
}

func upgradeCommand(args []string, item process) error {
	if len(args) != 1 {
		return usage()
	}
	_, err := install.Installer{Env: item.env}.Upgrade(context.Background(), args[0])
	return err
}

func runCommand(args []string, item process) error {
	if len(args) != 1 {
		return usage()
	}
	record, err := loadRecord(args[0])
	if err != nil {
		return err
	}
	requirements, err := install.DefaultRequirements()
	if err != nil {
		return err
	}
	if err := requireCurrentRecord(record, requirements); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := control.EnsureDaemon(ctx, record.Config, control.NewCommandRunner(item.env)); err != nil {
		return err
	}
	runner := newRunner(record, requirements, item)
	return runner.Run(ctx, time.Second)
}

func newRunner(record install.Record, requirements compatibility.Requirements, item process) transmit.Runner {
	counts := status.Counts{
		InstallID:     record.InstallID,
		DriverVersion: record.DriverVersion,
		ConfigDigest:  record.ConfigDigest,
	}
	invocation := invoke.New(record.Config.InvocationBaseURL, nil)
	return transmit.Runner{
		Config:     record.Config,
		Invocation: invocation,
		Control:    control.NewUnixClient(record.Config.ControlEndpoint),
		Counts:     &counts,
		Compatibility: compatibility.Verifier{
			Requirements: requirements,
			Invocation:   invocation,
			App: compatibility.CommandAppMetadata{
				CodexBin: record.Config.CodexBin,
				Env:      item.env,
			},
		},
	}
}

func requireCurrentRecord(record install.Record, requirements compatibility.Requirements) error {
	if record.StoredVersion() != requirements.InstallRecord.CurrentVersion {
		return compatibility.ErrInstallRecordUpgradeRequired
	}
	return nil
}

func statusCommand(args []string, item process) error {
	if len(args) != 1 {
		return usage()
	}
	record, err := loadRecord(args[0])
	if err != nil {
		fmt.Fprintln(item.stdout, status.RenderDiagnostic("local", compatibility.Code(err)))
		return err
	}
	fmt.Fprintln(item.stdout, status.Render(status.Counts{
		Source:        "local",
		InstallID:     record.InstallID,
		DriverVersion: record.DriverVersion,
		ConfigDigest:  record.ConfigDigest,
		LastErrorCode: record.LastDiagnosticCode,
	}))
	return nil
}

func doctorCommand(args []string, item process) error {
	if len(args) != 1 {
		return usage()
	}
	return doctor(args[0], nil, item)
}
