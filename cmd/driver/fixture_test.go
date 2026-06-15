package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/install"
)

type routeServer struct {
	*httptest.Server
	routes []string
}

func newRouteServer(t *testing.T) *routeServer {
	t.Helper()

	server := &routeServer{}
	server.Server = httptest.NewServer(http.HandlerFunc(server.handle))
	return server
}

func (s *routeServer) handle(writer http.ResponseWriter, request *http.Request) {
	s.routes = append(s.routes, request.URL.Path)
	switch request.URL.Path {
	case "/compatibility":
		writeJSON(writer, map[string]any{
			"contract_version":  "v1",
			"schema_set_digest": "sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac",
			"features": []string{
				"lease_claim",
				"lease_extend",
				"lease_ack",
				"lease_nack",
				"lease_capability",
				"declared_needs",
				"failure_payload",
			},
			"routes": []map[string]string{
				{"method": "POST", "path": "/claim"},
				{"method": "POST", "path": "/ack"},
				{"method": "POST", "path": "/nack"},
				{"method": "POST", "path": "/extend"},
				{"method": "GET", "path": "/compatibility"},
			},
		})
	default:
		http.NotFound(writer, request)
	}
}

func writeRecord(t *testing.T, baseURL string) string {
	t.Helper()

	home := t.TempDir()
	record := install.NewRecord(install.RecordInput{
		Config: config.Config{
			InvocationBaseURL: baseURL,
			ChannelKey:        "q01",
			LeaseSeconds:      60,
			CodexBin:          "/usr/bin/codex",
			CodexHome:         "/tmp/codex-home",
			ControlEndpoint:   "unix:///tmp/codex-home/codex.sock",
			WorkspaceRoot:     "/tmp/work",
			InputTextPointer:  "/work/prompt",
			CompletionNeeds:   []config.Need{{Kind: "done"}},
			FailureNeeds:      []config.Need{{Kind: "failed"}},
			RedactionMode:     "metadata_only",
		},
		Requirements:       requirementsFixture(),
		InvocationMetadata: metadataFixture(),
		AppServerDigest:    appDigest,
		LastDiagnosticCode: "compatible",
	})
	content, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "install.json"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	return home
}

func writeMalformedRecord(t *testing.T, home string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(home, "install.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readRecord(t *testing.T, home string) []byte {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(home, "install.json"))
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func commandProcess(stdout io.Writer) process {
	return process{
		env:    []string{},
		stdout: stdout,
		stderr: io.Discard,
	}
}

func writeJSON(writer http.ResponseWriter, value any) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(value)
}

func requireRoutes(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("routes = %v, want %v", got, want)
	}
	for index, value := range want {
		if got[index] != value {
			t.Fatalf("routes = %v, want %v", got, want)
		}
	}
}

func requireContains(t *testing.T, output string, want string) {
	t.Helper()

	if !strings.Contains(output, want) {
		t.Fatalf("output = %q, want %q", output, want)
	}
}

func requireLocalDiagnostic(t *testing.T, output string) {
	t.Helper()

	requireContains(t, output, "source=local")
	requireContains(t, output, "error_code=install_record_upgrade_required")
}

func requirementsFixture() compatibility.Requirements {
	return compatibility.Requirements{
		AcceptedVersions:         []string{"v1"},
		AcceptedSchemaSetDigests: []string{"sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac"},
		RequiredFeatures: []string{
			"lease_claim",
			"lease_extend",
			"lease_ack",
			"lease_nack",
			"lease_capability",
			"declared_needs",
			"failure_payload",
		},
		RequiredRoutes: []compatibility.Route{
			{Method: "POST", Path: "/claim"},
			{Method: "POST", Path: "/ack"},
			{Method: "POST", Path: "/nack"},
			{Method: "POST", Path: "/extend"},
			{Method: "GET", Path: "/compatibility"},
		},
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
	}
}

func metadataFixture() compatibility.Metadata {
	return compatibility.Metadata{
		ContractVersion: "v1",
		SchemaSetDigest: "sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac",
		Features: []string{
			"lease_claim",
			"lease_extend",
			"lease_ack",
			"lease_nack",
			"lease_capability",
			"declared_needs",
			"failure_payload",
		},
		Routes: []compatibility.Route{
			{Method: "POST", Path: "/claim"},
			{Method: "POST", Path: "/ack"},
			{Method: "POST", Path: "/nack"},
			{Method: "POST", Path: "/extend"},
			{Method: "GET", Path: "/compatibility"},
		},
	}
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

func requireCommands(t *testing.T, actual []string, expected []string) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf("commands = %v, want %v", actual, expected)
	}
	for index, value := range expected {
		if actual[index] != value {
			t.Fatalf("commands = %v, want %v", actual, expected)
		}
	}
}
