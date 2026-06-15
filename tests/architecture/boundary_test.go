package architecture

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCurrentTreeConforms(t *testing.T) {
	item := loadManifest(t, sourceRoot(t))

	requireTopLevelRoots(t, item)
	requireRequiredFiles(t, item)
	requireProductionPackages(t, item)
	requireSourceInventory(t, item)
}

func TestDeclaresPublicModulePath(t *testing.T) {
	const want = "github.com/pay-bye/agent-os-driver-codex-app-server"

	got := declaredModulePath(t, filepath.Join(sourceRoot(t), "go.mod"))

	if got != want {
		t.Fatalf("module path = %q, want %q", got, want)
	}
}

func TestExtractedModuleGateRuns(t *testing.T) {
	if os.Getenv("EXTRACTION_GATE") == "1" {
		t.Skip("nested extraction gate already running")
	}

	root := copyModule(t)
	command := exec.Command("bash", "./scripts/verify.sh", "--unit")
	command.Dir = root
	command.Env = append(os.Environ(), "EXTRACTION_GATE=1")
	output, err := command.CombinedOutput()

	if err != nil {
		t.Fatalf("extracted module gate failed: %v\n%s", err, output)
	}
}

func declaredModulePath(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	t.Fatal("module path not found")
	return ""
}

type manifest struct {
	SchemaVersion        int      `json:"schema_version"`
	RequiredFiles        []string `json:"required_files"`
	AllowedTopLevelRoots []string `json:"allowed_top_level_roots"`
	ProductionPackages   []string `json:"production_packages"`
	SourceInventory      []owner  `json:"source_inventory"`
}

type owner struct {
	Responsibility string   `json:"responsibility"`
	Path           string   `json:"path"`
	Files          []string `json:"files"`
}

func loadManifest(t *testing.T, root string) manifest {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, "quality", "boundary-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var item manifest
	if err := json.Unmarshal(content, &item); err != nil {
		t.Fatal(err)
	}
	if item.SchemaVersion != 1 {
		t.Fatalf("unexpected schema version %d", item.SchemaVersion)
	}
	return item
}

func requireTopLevelRoots(t *testing.T, item manifest) {
	t.Helper()

	entries, err := os.ReadDir(sourceRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if slices.Contains(item.AllowedTopLevelRoots, entry.Name()) {
			continue
		}
		t.Fatalf("unexpected top-level root %s", entry.Name())
	}
}

func requireRequiredFiles(t *testing.T, item manifest) {
	t.Helper()

	for _, path := range item.RequiredFiles {
		if _, err := os.Stat(filepath.Join(sourceRoot(t), filepath.FromSlash(path))); err != nil {
			t.Fatalf("required file missing %s: %v", path, err)
		}
	}
}

func requireProductionPackages(t *testing.T, item manifest) {
	t.Helper()

	for _, path := range item.ProductionPackages {
		if _, err := os.Stat(filepath.Join(sourceRoot(t), filepath.FromSlash(path))); err != nil {
			t.Fatalf("production package missing %s: %v", path, err)
		}
	}
}

func requireSourceInventory(t *testing.T, item manifest) {
	t.Helper()

	requireSourceInventoryEntryShape(t)
	owners := sourceOwners(item.SourceInventory)
	if len(owners) == 0 {
		t.Fatal("source inventory is empty")
	}
	violations := sourceInventoryViolations(t, sourceRoot(t), owners)
	if len(violations) > 0 {
		t.Fatalf("source file has no owner: %s", violations[0])
	}
}

func TestSourceInventoryRejectsUnlistedFilesInsideOwnedRoots(t *testing.T) {
	root := copyModule(t)
	writeFile(t, root, "internal/status/x99.go", "package status")
	writeFile(t, root, "internal/config/x98.go", "package config")

	item := loadManifest(t, root)
	violations := sourceInventoryViolations(t, root, sourceOwners(item.SourceInventory))

	requireUnowned(t, violations, "internal/status/x99.go")
	requireUnowned(t, violations, "internal/config/x98.go")
}

func sourceInventoryViolations(t *testing.T, root string, owners map[string]bool) []string {
	t.Helper()

	var violations []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative := relativePathFrom(t, root, path)
		if !sourceInventoryApplies(relative) || sourceOwned(relative, owners) {
			return nil
		}
		violations = append(violations, relative)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return violations
}

func requireSourceInventoryEntryShape(t *testing.T) {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(sourceRoot(t), "quality", "boundary-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(document["source_inventory"], &entries); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if len(entry) != 3 || entry["responsibility"] == nil || entry["path"] == nil || entry["files"] == nil {
			t.Fatalf("source inventory entry must contain only responsibility, path, and files: %#v", entry)
		}
	}
}

func sourceOwners(items []owner) map[string]bool {
	owners := map[string]bool{}
	for _, item := range items {
		root := filepath.ToSlash(filepath.Clean(item.Path))
		if item.Responsibility == "" || root == "." || len(item.Files) == 0 {
			continue
		}
		for _, file := range item.Files {
			owners[filepath.ToSlash(filepath.Join(root, filepath.FromSlash(file)))] = true
		}
	}
	return owners
}

func sourceInventoryApplies(path string) bool {
	switch path {
	case "CODE_OF_CONDUCT.md",
		"CONTRIBUTING.md",
		"GOVERNANCE.md",
		"LICENSE",
		"README.md",
		"SECURITY.md",
		"go.mod",
		"go.sum":
		return false
	default:
		return true
	}
}

func sourceOwned(path string, owners map[string]bool) bool {
	return owners[path]
}

func requireUnowned(t *testing.T, violations []string, path string) {
	t.Helper()

	if !slices.Contains(violations, path) {
		t.Fatalf("violations = %v, want %s", violations, path)
	}
}

func writeFile(t *testing.T, root string, path string, content string) {
	t.Helper()

	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyModule(t *testing.T) string {
	t.Helper()

	source := sourceRoot(t)
	target := filepath.Join(t.TempDir(), "driver")
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative := relativePath(t, path)
		return copyEntry(path, filepath.Join(target, filepath.FromSlash(relative)), entry)
	})
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func copyEntry(source string, target string, entry os.DirEntry) error {
	if entry.IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(target, content, modeFor(entry))
}

func modeFor(entry os.DirEntry) os.FileMode {
	info, err := entry.Info()
	if err != nil {
		return 0o644
	}
	return info.Mode()
}

func relativePath(t *testing.T, path string) string {
	t.Helper()

	return relativePathFrom(t, sourceRoot(t), path)
}

func relativePathFrom(t *testing.T, root string, path string) string {
	t.Helper()

	relative, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(relative)
}

func sourceRoot(t *testing.T) string {
	t.Helper()

	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return findRoot(t, working)
}

func findRoot(t *testing.T, start string) string {
	t.Helper()

	current := start
	for {
		if fileExists(current, "go.mod") && fileExists(current, "quality/boundary-manifest.json") {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatal("source root not found")
		}
		current = parent
	}
}

func fileExists(root string, path string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	return err == nil
}
