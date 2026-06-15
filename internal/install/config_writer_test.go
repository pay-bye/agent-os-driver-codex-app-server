package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteConfigFilesMaterializesRunnerRegistration(t *testing.T) {
	home := t.TempDir()

	if err := writeConfigFiles(home); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(home, "runner", "registration.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"registration": "portable-wrapper"`) {
		t.Fatalf("registration content = %s", content)
	}
}
