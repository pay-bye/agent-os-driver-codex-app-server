package install

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
)

func TestReadParsesSourceRequirements(t *testing.T) {
	requirements, err := ReadRequirements(sourceManifest(t))
	if err != nil {
		t.Fatal(err)
	}

	requireEqual(t, requirements.AcceptedVersions, []string{"v1"})
	requireEqual(t, requirements.AcceptedSchemaSetDigests, []string{
		"sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac",
	})
	requireEqual(t, requirements.RequiredFeatures, []string{
		"lease_claim",
		"lease_extend",
		"lease_ack",
		"lease_nack",
		"lease_capability",
		"declared_needs",
		"failure_payload",
	})
	requireEqual(t, requirements.RequiredRoutes, []compatibility.Route{
		{Method: "POST", Path: "/claim"},
		{Method: "POST", Path: "/ack"},
		{Method: "POST", Path: "/nack"},
		{Method: "POST", Path: "/extend"},
		{Method: "GET", Path: "/compatibility"},
	})
	if requirements.InstallRecord.CurrentVersion != 3 {
		t.Fatalf("current version = %d, want 3", requirements.InstallRecord.CurrentVersion)
	}
	if requirements.InstallRecord.MinimumReadableVersion != 1 {
		t.Fatalf("minimum readable version = %d, want 1", requirements.InstallRecord.MinimumReadableVersion)
	}
	if requirements.CodexCLI.Version.Evidence != "record_current" {
		t.Fatalf("version evidence = %q, want record_current", requirements.CodexCLI.Version.Evidence)
	}
	if requirements.AppServer.SchemaDigestCanonicalization != "json_sort_keys_v1" {
		t.Fatalf("canonicalization = %q, want json_sort_keys_v1", requirements.AppServer.SchemaDigestCanonicalization)
	}
	requireEqual(t, requirements.AppServer.RequiredSchemaFiles, requiredSchemaFiles())
	requireEqual(t,
		requirements.AppServer.RequiredMethods,
		[]string{"initialize", "thread/start", "turn/start", "turn/interrupt"},
	)
	requireEqual(t, requirements.AppServer.RequiredNotifications, []string{"turn/completed"})
	if requirements.AppServer.ControlSurface != "uds_websocket" {
		t.Fatalf("control surface = %q, want uds_websocket", requirements.AppServer.ControlSurface)
	}
}

func TestDefaultRequirementsMatchSourceManifest(t *testing.T) {
	fromSource, err := ReadRequirements(sourceManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	fromEmbed, err := DefaultRequirements()
	if err != nil {
		t.Fatal(err)
	}

	requireEqual(t, fromEmbed, fromSource)
}

func TestReadRejectsMalformedRequirements(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing compatibility", body: `{"schema_version":1}`},
		{name: "contract version format", body: strings.Replace(validManifest, `"v1"`, `"x91"`, 1)},
		{name: "schema canonicalization", body: strings.Replace(validManifest, "json_sort_keys_v1", "x91", 1)},
		{name: "feature", body: strings.Replace(validManifest, `"lease_claim"`, `"x91"`, 1)},
		{name: "route", body: strings.Replace(validManifest, `"/claim"`, `"claim"`, 1)},
		{name: "unknown route", body: strings.Replace(validManifest, `"/claim"`, `"/unknown"`, 1)},
		{name: "duplicate route", body: strings.Replace(validManifest, `"/compatibility"`, `"/claim"`, 1)},
		{
			name: "record version",
			body: strings.Replace(
				validManifest,
				`"minimum_readable_version": 1`,
				`"minimum_readable_version": 4`,
				1,
			),
		},
		{name: "app method", body: strings.Replace(validManifest, `"thread/start"`, `""`, 1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ReadRequirements(writeManifest(t, test.body))

			if !errors.Is(err, compatibility.ErrMetadataMalformed) {
				t.Fatalf("error = %v, want metadata_malformed", err)
			}
		})
	}
}

func TestRecordVersionRejectsUnreadableRecords(t *testing.T) {
	requirements := compatibility.Requirements{
		InstallRecord: compatibility.RecordRequirement{
			CurrentVersion:         3,
			MinimumReadableVersion: 1,
		},
	}

	if err := requirements.CheckRecordVersion(1); err != nil {
		t.Fatalf("version 1 rejected: %v", err)
	}
	if err := requirements.CheckRecordVersion(2); err != nil {
		t.Fatalf("version 2 rejected: %v", err)
	}
	if err := requirements.CheckRecordVersion(3); err != nil {
		t.Fatalf("version 3 rejected: %v", err)
	}
	if !errors.Is(requirements.CheckRecordVersion(0), compatibility.ErrInstallRecordUpgradeRequired) {
		t.Fatal("version 0 did not require upgrade")
	}
	if !errors.Is(requirements.CheckRecordVersion(4), compatibility.ErrInstallRecordUpgradeRequired) {
		t.Fatal("version 4 did not require upgrade")
	}
}

func sourceManifest(t *testing.T) string {
	t.Helper()

	return filepath.Join(sourceRoot(t), "internal", "install", "driver_manifest.json")
}

func sourceRoot(t *testing.T) string {
	t.Helper()

	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if exists(filepath.Join(current, "go.mod")) && exists(filepath.Join(current, "quality", "boundary-manifest.json")) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatal("source root not found")
		}
		current = parent
	}
}

func writeManifest(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func requireEqual[T any](t *testing.T, got T, want T) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

const validManifest = `{
  "schema_version": 1,
  "compatibility": {
    "accepted_versions": ["v1"],
    "accepted_schema_set_digests": [
      "sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac"
    ],
    "required_features": [
      "lease_claim",
      "lease_extend",
      "lease_ack",
      "lease_nack",
      "lease_capability",
      "declared_needs",
      "failure_payload"
    ],
    "required_routes": [
      {"method": "POST", "path": "/claim"},
      {"method": "POST", "path": "/ack"},
      {"method": "POST", "path": "/nack"},
      {"method": "POST", "path": "/extend"},
      {"method": "GET", "path": "/compatibility"}
    ],
    "install_record": {
      "current_version": 3,
      "minimum_readable_version": 1
    },
    "codex_cli": {
      "version": {
        "evidence": "record_current"
      }
    },
    "app_server": {
      "schema_digest_canonicalization": "json_sort_keys_v1",
      "required_schema_files": [
        "codex_app_server_protocol.schemas.json",
        "codex_app_server_protocol.v2.schemas.json",
        "v2/ThreadStartParams.json",
        "v2/ThreadStartResponse.json",
        "v2/TurnStartParams.json",
        "v2/TurnStartResponse.json",
        "v2/TurnCompletedNotification.json",
        "v2/TurnInterruptParams.json",
        "v2/TurnInterruptResponse.json",
        "v2/RemoteControlEnableResponse.json",
        "v2/RemoteControlDisableResponse.json",
        "v2/RemoteControlStatusReadResponse.json",
        "v2/RemoteControlStatusChangedNotification.json"
      ],
      "required_methods": ["initialize", "thread/start", "turn/start", "turn/interrupt"],
      "required_notifications": ["turn/completed"],
      "control_surface": "uds_websocket"
    }
  }
}
`
