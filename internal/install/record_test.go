package install

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
)

func TestRecordUsesOpaqueInstallID(t *testing.T) {
	record := NewRecord(recordInput(testConfig("http://127.0.0.1:8080")))

	requireHexLength(t, record.InstallID, 24)
}

func TestRecordRoundTripStoresCompatibilityEvidence(t *testing.T) {
	item := testConfig("http://127.0.0.1:8080")
	input := recordInput(item)
	record := NewRecord(input)

	home := t.TempDir()
	if err := writeRecord(home, record); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(home)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.StoredVersion() != 2 {
		t.Fatalf("stored version = %d, want 2", loaded.StoredVersion())
	}
	if loaded.RedactionMode != "metadata_only" {
		t.Fatalf("redaction mode = %q, want metadata_only", loaded.RedactionMode)
	}
	if loaded.LastDiagnosticCode != "compatible" {
		t.Fatalf("diagnostic = %q, want compatible", loaded.LastDiagnosticCode)
	}
	if !reflect.DeepEqual(loaded.Compatibility.Requirements, input.Requirements) {
		t.Fatalf("requirements = %#v, want %#v", loaded.Compatibility.Requirements, input.Requirements)
	}
	if !reflect.DeepEqual(loaded.Compatibility.InvocationMetadata, input.InvocationMetadata) {
		t.Fatalf("metadata = %#v, want %#v", loaded.Compatibility.InvocationMetadata, input.InvocationMetadata)
	}
	if loaded.Compatibility.AppServerDigest != input.AppServerDigest {
		t.Fatalf("app digest = %q, want %q", loaded.Compatibility.AppServerDigest, input.AppServerDigest)
	}
}

func recordInput(item config.Config) RecordInput {
	appDigest := "sha256:12e9f18872355d73f0767a5ef52cf5c276be7e6d37d88a4b684cd58c46698e0c"
	requirements := compatibility.Requirements{
		AcceptedVersions:         []string{"v1"},
		AcceptedSchemaSetDigests: []string{"sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac"},
		RequiredFeatures:         []string{"lease_claim", "lease_capability"},
		RequiredRoutes:           routeInventoryFixture(),
		InstallRecord: compatibility.RecordRequirement{
			CurrentVersion:         2,
			MinimumReadableVersion: 1,
		},
		AppServer: compatibility.AppRequirement{
			RequiredMethods:       []string{"thread/start"},
			RequiredNotifications: []string{"turn/completed"},
		},
	}
	metadata := compatibility.Metadata{
		ContractVersion: "v1",
		SchemaSetDigest: "sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac",
		Features:        []string{"lease_claim", "lease_capability"},
		Routes:          routeInventoryFixture(),
	}
	return RecordInput{
		Config:             item,
		Requirements:       requirements,
		InvocationMetadata: metadata,
		AppServerDigest:    appDigest,
		LastDiagnosticCode: "compatible",
	}
}

func TestRecordWithoutVersionReadsAsVersionOne(t *testing.T) {
	record := Record{}

	if record.StoredVersion() != 1 {
		t.Fatalf("stored version = %d, want 1", record.StoredVersion())
	}
}

func testConfig(baseURL string) config.Config {
	return config.Config{
		InvocationBaseURL: baseURL,
		ChannelKey:        "q01",
		LeaseSeconds:      60,
		CodexBin:          "/usr/bin/codex",
		CodexHome:         "/tmp/codex-home",
		ControlEndpoint:   "unix:///tmp/codex-home/codex-app-server.sock",
		WorkspaceRoot:     "/tmp/work",
		InputTextPointer:  "/work/prompt",
		CompletionNeeds:   []config.Need{{Kind: "done"}},
		FailureNeeds:      []config.Need{{Kind: "failed"}},
		RedactionMode:     "metadata_only",
	}
}

func requireHexLength(t *testing.T, value string, length int) {
	t.Helper()

	pattern := regexp.MustCompile(`^[0-9a-f]+$`)
	if len(value) != length || !pattern.MatchString(value) {
		t.Fatalf("id = %q, want %d lowercase hex chars", value, length)
	}
}
