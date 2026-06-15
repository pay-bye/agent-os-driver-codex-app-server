package install

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
)

func applyInstalled(t *testing.T, path string, home string) (Record, error) {
	t.Helper()

	return Installer{
		CodexCheck: acceptCodex,
		Verifier:   &fakeVerifier{result: resultFixture()},
	}.Apply(context.Background(), path, home)
}

func writeConfig(t *testing.T, baseURL string) string {
	t.Helper()

	home := filepath.Join(t.TempDir(), "codex-home")
	item := config.Config{
		InvocationBaseURL: baseURL,
		ChannelKey:        "q01",
		LeaseSeconds:      60,
		CodexBin:          "/usr/bin/codex",
		CodexHome:         home,
		ControlEndpoint:   "unix://" + filepath.Join(home, "codex-app-server.sock"),
		WorkspaceRoot:     "/tmp/work",
		InputTextPointer:  "/work/prompt",
		CompletionNeeds:   []config.Need{{Kind: "done"}},
		FailureNeeds:      []config.Need{{Kind: "failed"}},
		RedactionMode:     "metadata_only",
	}
	content, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readInstallRecord(t *testing.T, home string) []byte {
	t.Helper()

	content, err := os.ReadFile(recordPath(home))
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func requireFiles(t *testing.T, root string, expected []string) {
	t.Helper()

	for _, path := range expected {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			t.Fatalf("expected file %s: %v", path, err)
		}
	}
}

func requireMissing(t *testing.T, root string, path string) {
	t.Helper()

	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err == nil {
		t.Fatalf("expected missing file %s", path)
	}
}

func acceptCodex(string) error {
	return nil
}

type fakeVerifier struct {
	result compatibility.Result
	err    error
	calls  int
}

func (v *fakeVerifier) Verify(context.Context, config.Config) (compatibility.Result, error) {
	v.calls++
	if v.err != nil {
		return compatibility.Result{DiagnosticCode: compatibility.Code(v.err)}, v.err
	}
	return v.result, nil
}

func resultFixture() compatibility.Result {
	return compatibility.Result{
		Requirements: compatibility.Requirements{
			AcceptedVersions:         []string{"v1"},
			AcceptedSchemaSetDigests: []string{"sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac"},
			RequiredFeatures:         []string{"lease_claim", "lease_capability"},
			RequiredRoutes:           routeInventoryFixture(),
			InstallRecord: compatibility.RecordRequirement{
				CurrentVersion:         3,
				MinimumReadableVersion: 1,
			},
			CodexCLI: compatibility.CLIRequirement{
				Version: compatibility.VersionRequirement{Evidence: "record_current"},
			},
			AppServer: compatibility.AppRequirement{
				SchemaDigestCanonicalization: "json_sort_keys_v1",
				RequiredSchemaFiles:          requiredSchemaFiles(),
				RequiredMethods:              []string{"initialize", "thread/start", "turn/start", "turn/interrupt"},
				RequiredNotifications:        []string{"turn/completed"},
				ControlSurface:               "uds_websocket",
			},
		},
		InvocationMetadata: compatibility.Metadata{
			ContractVersion: "v1",
			SchemaSetDigest: "sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac",
			Features:        []string{"lease_claim", "lease_capability"},
			Routes:          routeInventoryFixture(),
		},
		AppServerDigest: appDigest,
		DiagnosticCode:  "compatible",
	}
}

func routeInventoryFixture() []compatibility.Route {
	return []compatibility.Route{
		{Method: "POST", Path: "/claim"},
		{Method: "POST", Path: "/ack"},
		{Method: "POST", Path: "/nack"},
		{Method: "POST", Path: "/extend"},
		{Method: "GET", Path: "/compatibility"},
	}
}

func requireError(t *testing.T, err error, expected string) {
	t.Helper()

	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error containing %q, got %v", expected, err)
	}
}

func sorted(values []string) []string {
	slices.Sort(values)
	return values
}

const currentVersion = "codex-cli test-current"
const appDigest = "sha256:588e3e4ad47defffca22dba7769d0c902756165931eedee139e6e4f9efdac16d"

func requiredSchemaFiles() []string {
	return []string{
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
		"v2/RemoteControlStatusChangedNotification.json",
	}
}
