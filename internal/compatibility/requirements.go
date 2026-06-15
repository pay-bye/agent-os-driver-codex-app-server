package compatibility

import (
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/invoke"
)

type Requirements struct {
	AcceptedVersions         []string          `json:"accepted_versions"`
	AcceptedSchemaSetDigests []string          `json:"accepted_schema_set_digests"`
	RequiredFeatures         []string          `json:"required_features"`
	RequiredRoutes           []Route           `json:"required_routes"`
	InstallRecord            RecordRequirement `json:"install_record"`
	CodexCLI                 CLIRequirement    `json:"codex_cli"`
	AppServer                AppRequirement    `json:"app_server"`
}

func (r Requirements) CheckMetadata(metadata Metadata) error {
	if err := metadata.Validate(); err != nil {
		return err
	}
	if err := r.CheckContractVersion(metadata.ContractVersion); err != nil {
		return err
	}
	if err := requireDigest(metadata.SchemaSetDigest); err != nil {
		return err
	}
	if !contains(r.AcceptedSchemaSetDigests, metadata.SchemaSetDigest) {
		return ErrSchemaDigestUnaccepted
	}
	if err := requireFeatures(metadata.Features); err != nil {
		return err
	}
	if !containsAll(metadata.Features, r.RequiredFeatures) {
		return ErrFeatureMissing
	}
	if !invoke.ContainsRoutes(metadata.Routes, r.RequiredRoutes) {
		return ErrRouteMissing
	}
	return nil
}

func (r Requirements) CheckContractVersion(version string) error {
	if contains(r.AcceptedVersions, version) {
		return nil
	}
	got, err := versionNumber(version)
	if err != nil {
		return ErrMetadataMalformed
	}
	minimum, maximum := versionBounds(r.AcceptedVersions)
	if got < minimum {
		return ErrContractTooOld
	}
	if got > maximum {
		return ErrContractTooNew
	}
	return ErrContractTooNew
}

func (r Requirements) CheckApp(metadata AppMetadata) error {
	if !digestPattern.MatchString(metadata.SchemaDigest) {
		return ErrAppProtocolDrift
	}
	if !containsAll(metadata.SchemaFiles, r.AppServer.RequiredSchemaFiles) {
		return ErrAppProtocolDrift
	}
	if !containsAll(metadata.Methods, r.AppServer.RequiredMethods) {
		return ErrAppProtocolDrift
	}
	if !containsAll(metadata.Notifications, r.AppServer.RequiredNotifications) {
		return ErrAppProtocolDrift
	}
	if metadata.ControlSurface != r.AppServer.ControlSurface {
		return ErrAppProtocolDrift
	}
	return nil
}

func (r Requirements) Validate() error {
	checks := []func() error{
		func() error { return requireVersions(r.AcceptedVersions) },
		func() error { return requireDigests(r.AcceptedSchemaSetDigests) },
		func() error { return requireFeatures(r.RequiredFeatures) },
		func() error { return requireRoutes(r.RequiredRoutes) },
		func() error { return requireRecord(r.InstallRecord) },
		func() error { return requireCLI(r.CodexCLI) },
		func() error { return requireApp(r.AppServer) },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func (r Requirements) CheckRecordVersion(version int) error {
	if version < r.InstallRecord.MinimumReadableVersion || version > r.InstallRecord.CurrentVersion {
		return ErrInstallRecordUpgradeRequired
	}
	return nil
}

type CLIRequirement struct {
	Version VersionRequirement `json:"version"`
}

type VersionRequirement struct {
	Evidence string `json:"evidence"`
}

type RecordRequirement struct {
	CurrentVersion         int `json:"current_version"`
	MinimumReadableVersion int `json:"minimum_readable_version"`
}

type AppRequirement struct {
	SchemaDigestCanonicalization string   `json:"schema_digest_canonicalization"`
	RequiredSchemaFiles          []string `json:"required_schema_files"`
	RequiredMethods              []string `json:"required_methods"`
	RequiredNotifications        []string `json:"required_notifications"`
	ControlSurface               string   `json:"control_surface"`
}
