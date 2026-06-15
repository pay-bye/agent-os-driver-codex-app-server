package install

import (
	_ "embed"
	"path/filepath"
)

func writeConfigFiles(home string) error {
	return writeFile(filepath.Join(home, "runner", "registration.json"), []byte(runnerContent()))
}

func runnerContent() string {
	return `{
  "registration": "portable-wrapper",
  "owned_by": "codex-app-server"
}
`
}
