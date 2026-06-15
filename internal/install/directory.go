package install

import (
	_ "embed"
	"os"
	"path/filepath"
)

func Remove(home string) error {
	for _, path := range ownedPaths(home) {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func ownedPaths(home string) []string {
	return []string{
		recordPath(home),
		filepath.Join(home, "protocol"),
		filepath.Join(home, "runner"),
	}
}
