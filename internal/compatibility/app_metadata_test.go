package compatibility

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandAppMetadataReadsGeneratedProtocolMetadata(t *testing.T) {
	reader := CommandAppMetadata{
		CodexBin: "codex",
		Run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if len(args) == 1 && args[0] == "--version" {
				return []byte(currentVersion + "\n"), nil
			}
			root := outputRoot(t, args)
			writeGeneratedSchemas(t, root)
			return nil, nil
		},
	}

	metadata, err := reader.Metadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if metadata.CodexVersion != currentVersion {
		t.Fatalf("version = %q, want %q", metadata.CodexVersion, currentVersion)
	}
	if !strings.HasPrefix(metadata.SchemaDigest, "sha256:") {
		t.Fatalf("digest = %q, want sha256", metadata.SchemaDigest)
	}
	requireEqual(t, metadata.SchemaFiles, requiredSchemaFiles())
	requireEqual(t, metadata.Methods, []string{"initialize", "thread/start", "turn/start", "turn/interrupt"})
	requireEqual(t, metadata.Notifications, []string{"turn/completed"})
}

func TestCommandAppMetadataRecordsCurrentVersion(t *testing.T) {
	reader := CommandAppMetadata{
		CodexBin: "codex",
		Run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if len(args) == 1 && args[0] == "--version" {
				return []byte("codex-cli 0.133.0\n"), nil
			}
			root := outputRoot(t, args)
			writeGeneratedSchemas(t, root)
			return nil, nil
		},
	}

	metadata, err := reader.Metadata(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if metadata.CodexVersion != "codex-cli 0.133.0" {
		t.Fatalf("version = %q, want current version", metadata.CodexVersion)
	}
}

func TestSchemaDigestCanonicalizesJSONObjects(t *testing.T) {
	left := t.TempDir()
	right := t.TempDir()
	writeGeneratedSchema(t, left, "v2/ThreadStartParams.json", `{"b":2,"a":1}`)
	writeGeneratedSchema(t, right, "v2/ThreadStartParams.json", `{"a":1,"b":2}`)

	leftDigest, err := schemaDigest(left)
	if err != nil {
		t.Fatal(err)
	}
	rightDigest, err := schemaDigest(right)
	if err != nil {
		t.Fatal(err)
	}

	if leftDigest != rightDigest {
		t.Fatalf("canonical digests differ: %s != %s", leftDigest, rightDigest)
	}
}

func TestCommandAppMetadataMatchesInstalledCodex(t *testing.T) {
	metadata, err := CommandAppMetadata{
		CodexBin: "codex",
	}.Metadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if metadata.CodexVersion == "" {
		t.Fatal("version missing")
	}
	if !strings.HasPrefix(metadata.SchemaDigest, "sha256:") {
		t.Fatalf("schema digest = %q, want sha256 evidence", metadata.SchemaDigest)
	}
}

func TestCommandAppMetadataRejectsMissingGeneratedSchema(t *testing.T) {
	reader := CommandAppMetadata{
		CodexBin: "codex",
		Run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if len(args) == 1 && args[0] == "--version" {
				return []byte(currentVersion + "\n"), nil
			}
			root := outputRoot(t, args)
			writeGeneratedSchema(t, root, "codex_app_server_protocol.v2.schemas.json", `{}`)
			return nil, nil
		},
	}

	_, err := reader.Metadata(context.Background())

	if err != ErrAppProtocolDrift {
		t.Fatalf("error = %v, want app_protocol_drift", err)
	}
}

func outputRoot(t *testing.T, args []string) string {
	t.Helper()

	for index, arg := range args {
		if arg == "--out" && index+1 < len(args) {
			return args[index+1]
		}
	}
	t.Fatal("missing output path")
	return ""
}

func writeGeneratedSchemas(t *testing.T, root string) {
	t.Helper()

	for _, path := range requiredSchemaFiles() {
		writeGeneratedSchema(t, root, path, `{"type":"object"}`)
	}
}

func writeGeneratedSchema(t *testing.T, root string, path string, content string) {
	t.Helper()

	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
