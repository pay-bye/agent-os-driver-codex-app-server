package install

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
	"os"
	"path/filepath"
	"time"
)

const DriverVersion = "0.1.0"

type Record struct {
	Version            int                           `json:"install_record_version"`
	InstallID          string                        `json:"install_id"`
	DriverVersion      string                        `json:"driver_version"`
	ConfigDigest       string                        `json:"config_digest"`
	RedactionMode      string                        `json:"redaction_mode"`
	LastDiagnosticCode string                        `json:"last_diagnostic_code"`
	InstalledAt        time.Time                     `json:"installed_at"`
	Config             config.Config                 `json:"config"`
	Compatibility      compatibility.InstallEvidence `json:"compatibility"`
}

func NewRecord(input RecordInput) Record {
	digest := configDigest(input.Config)
	return Record{
		Version:            input.Requirements.InstallRecord.CurrentVersion,
		InstallID:          digest[:24],
		DriverVersion:      DriverVersion,
		ConfigDigest:       digest,
		RedactionMode:      input.Config.RedactionMode,
		LastDiagnosticCode: input.LastDiagnosticCode,
		InstalledAt:        time.Now().UTC(),
		Config:             input.Config,
		Compatibility: compatibility.InstallEvidence{
			Requirements:       input.Requirements,
			InvocationMetadata: input.InvocationMetadata,
			AppServerDigest:    input.AppServerDigest,
		},
	}
}

func (r Record) StoredVersion() int {
	if r.Version == 0 {
		return 1
	}
	return r.Version
}

type RecordInput struct {
	Config             config.Config
	Requirements       compatibility.Requirements
	InvocationMetadata compatibility.Metadata
	AppServerDigest    string
	LastDiagnosticCode string
}

func Load(home string) (Record, error) {
	file, err := os.Open(recordPath(home))
	if err != nil {
		return Record{}, err
	}
	defer file.Close()

	var record Record
	if err := json.NewDecoder(file).Decode(&record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func LoadReadable(home string) (Record, error) {
	record, err := Load(home)
	if err != nil {
		return Record{}, compatibility.ErrInstallRecordUpgradeRequired
	}
	return record, nil
}

func writeRecord(home string, record Record) error {
	content, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(recordPath(home), content)
}

func recordPath(home string) string {
	return filepath.Join(home, "install.json")
}

func configDigest(item config.Config) string {
	content, err := json.Marshal(item)
	if err != nil {
		panic(fmt.Sprintf("config digest encode failed: %v", err))
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
