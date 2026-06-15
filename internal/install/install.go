package install

import (
	"context"
	_ "embed"
	"errors"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
	"os"
)

type Verifier interface {
	Verify(context.Context, config.Config) (compatibility.Result, error)
}

type Installer struct {
	CodexCheck config.CodexCheck
	Verifier   Verifier
	Env        []string
}

func (i Installer) Apply(ctx context.Context, path string, home string) (Record, error) {
	item, err := config.Read(path, codexCheck(i.CodexCheck))
	if err != nil {
		return Record{}, err
	}
	result, err := i.verifier().Verify(ctx, item)
	if err != nil {
		return Record{}, err
	}
	record := NewRecord(RecordInput{
		Config:             item,
		Requirements:       result.Requirements,
		InvocationMetadata: result.InvocationMetadata,
		AppServerDigest:    result.AppServerDigest,
		LastDiagnosticCode: result.DiagnosticCode,
	})
	existing, err := Load(home)
	if err == nil {
		return existingOrConflict(existing, record)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Record{}, err
	}
	if err := writeRecord(home, record); err != nil {
		return Record{}, err
	}
	if err := writeOwnedFiles(home); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (i Installer) verifier() Verifier {
	if i.Verifier != nil {
		return i.Verifier
	}
	return liveVerifier{Env: i.Env}
}

func (i Installer) Upgrade(ctx context.Context, home string) (Record, error) {
	existing, err := LoadReadable(home)
	if err != nil {
		return Record{}, err
	}
	requirements, err := DefaultRequirements()
	if err != nil {
		return Record{}, err
	}
	if err := requirements.CheckRecordVersion(existing.StoredVersion()); err != nil {
		return Record{}, err
	}
	if err := requireCurrentConfig(existing.Config); err != nil {
		return Record{}, err
	}
	result, err := i.verifier().Verify(ctx, existing.Config)
	if err != nil {
		return Record{}, err
	}
	record := NewRecord(RecordInput{
		Config:             existing.Config,
		Requirements:       result.Requirements,
		InvocationMetadata: result.InvocationMetadata,
		AppServerDigest:    result.AppServerDigest,
		LastDiagnosticCode: result.DiagnosticCode,
	})
	if err := writeRecord(home, record); err != nil {
		return Record{}, err
	}
	if err := writeOwnedFiles(home); err != nil {
		return Record{}, err
	}
	return record, nil
}

func Apply(path string, home string, check config.CodexCheck) (Record, error) {
	return Installer{CodexCheck: check}.Apply(context.Background(), path, home)
}

func requireCurrentConfig(item config.Config) error {
	if err := item.Validate(nil); err != nil {
		return compatibility.ErrInstallRecordUpgradeRequired
	}
	return nil
}

func existingOrConflict(existing Record, next Record) (Record, error) {
	if existing.ConfigDigest == next.ConfigDigest {
		return existing, nil
	}
	return Record{}, errors.New("install_conflict")
}
