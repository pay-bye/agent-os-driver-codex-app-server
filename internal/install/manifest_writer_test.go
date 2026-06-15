package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteManifestFilesMaterializesRequiredSchemas(t *testing.T) {
	home := t.TempDir()

	if err := writeManifestFiles(home); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(home, "protocol", "required-schemas.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"v2/ThreadStartParams.json"`) {
		t.Fatalf("schema content = %s", content)
	}
}
