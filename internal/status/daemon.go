package status

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/control"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	codePrerequisiteMissing = "daemon_readiness_prerequisite_missing"
	codePathEscape          = "daemon_readiness_path_escape"
	codeCleanupFailed       = "daemon_readiness_cleanup_failed"
)

type DaemonCommand struct {
	Name string
	Args []string
	Env  []string
}

type DaemonRunner func(context.Context, DaemonCommand) ([]byte, error)

type ThreadStarter func(context.Context, string, string) (string, error)

type ProcessReader func(int, string) (ProcessEvidence, error)

type Killer func(int) error

type ProcessEvidence struct {
	Alive      bool
	Command    string
	Executable string
	Env        []string
}

type DaemonOptions struct {
	BinaryOverride string
	LookupPath     func(string) (string, error)
	Runner         DaemonRunner
	StartThread    ThreadStarter
	ReadProcess    ProcessReader
	Kill           Killer
	RemoveAll      func(string) error
}

type DaemonResult struct {
	BinaryPath string
	Version    string
	Runtime    time.Duration
	ThreadID   string
	TempClass  string
	Cleanup    string
	Commands   []string
	PathFields []string
}

type session struct {
	binary        string
	version       string
	root          string
	home          string
	codexHome     string
	workspace     string
	runner        DaemonRunner
	threadStarter ThreadStarter
	readProcess   ProcessReader
	kill          Killer
	removeAll     func(string) error
	commands      []string
	pathFields    []string
	socket        string
	pidFile       string
	pid           int
	threadID      string
}

func CheckDaemon(ctx context.Context, options DaemonOptions) (DaemonResult, error) {
	started := time.Now()
	current, err := newSession(ctx, options)
	if err != nil {
		return DaemonResult{}, err
	}
	removeRoot := true
	defer func() {
		if removeRoot {
			_ = current.removeAll(current.root)
		}
	}()

	if err := seedManagedPath(current); err != nil {
		return daemonResult(current, started, "not_started"), err
	}
	cleanup := "not_started"

	if err := startDaemon(current, ctx); err != nil {
		if daemonStarted(current) {
			cleanup = cleanupDaemon(current, ctx)
			removeRoot = cleanup == "stopped"
		}
		return daemonResult(current, started, cleanup), err
	}
	if err := checkDaemonVersion(current, ctx); err != nil {
		cleanup = cleanupDaemon(current, ctx)
		removeRoot = cleanup == "stopped"
		return daemonResult(current, started, cleanup), err
	}
	if err := startDaemonThread(current, ctx); err != nil {
		cleanup = cleanupDaemon(current, ctx)
		removeRoot = cleanup == "stopped"
		return daemonResult(current, started, cleanup), err
	}
	cleanup = cleanupDaemon(current, ctx)
	if cleanup != "stopped" {
		removeRoot = false
		return daemonResult(current, started, cleanup), fmt.Errorf("%s: %s", codeCleanupFailed, cleanup)
	}
	return daemonResult(current, started, cleanup), nil
}

func createDirs(current *session) error {
	for _, path := range []string{
		current.home,
		current.codexHome,
		filepath.Join(current.root, "tmp"),
		filepath.Join(current.root, "xdg-config"),
		filepath.Join(current.root, "xdg-cache"),
		filepath.Join(current.root, "xdg-runtime"),
		current.workspace,
	} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func seedManagedPath(current *session) error {
	path := filepath.Join(current.codexHome, "packages", "standalone", "current", "codex")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.Symlink(current.binary, path); err == nil {
		return nil
	}
	return copyFile(current.binary, path)
}

func startDaemon(current *session, ctx context.Context) error {
	response, err := runDaemonCommand(current, ctx, "start")
	if err != nil {
		return err
	}
	if err := readDaemonResponse(current, response); err != nil {
		return err
	}
	return discoverPidFile(current)
}

func checkDaemonVersion(current *session, ctx context.Context) error {
	response, err := runDaemonCommand(current, ctx, "version")
	if err != nil {
		return err
	}
	return readDaemonResponse(current, response)
}

func runDaemonCommand(current *session, ctx context.Context, action string) ([]byte, error) {
	current.commands = append(current.commands, action)
	return current.runner(ctx, DaemonCommand{
		Name: current.binary,
		Args: []string{"app-server", "daemon", action},
		Env:  commandEnvironment(sessionEnv(current)),
	})
}

func sessionEnv(current *session) []string {
	return []string{
		"HOME=" + current.home,
		"CODEX_HOME=" + current.codexHome,
		"TMPDIR=" + filepath.Join(current.root, "tmp"),
		"XDG_CONFIG_HOME=" + filepath.Join(current.root, "xdg-config"),
		"XDG_CACHE_HOME=" + filepath.Join(current.root, "xdg-cache"),
		"XDG_RUNTIME_DIR=" + filepath.Join(current.root, "xdg-runtime"),
	}
}

func readDaemonResponse(current *session, content []byte) error {
	var response map[string]any
	if err := json.Unmarshal(content, &response); err != nil {
		return err
	}
	fields := pathFields(response)
	for name, value := range fields {
		path := daemonPath(current, value)
		if !inside(path, current.codexHome) {
			return fmt.Errorf("%s: %s", codePathEscape, name)
		}
		current.pathFields = append(current.pathFields, name)
		if current.socket == "" && strings.Contains(strings.ToLower(name), "socket") {
			current.socket = path
		}
		if current.pidFile == "" && namesPidFile(name) {
			current.pidFile = path
		}
	}
	for name, value := range response {
		if current.pid == 0 && namesPid(name) {
			current.pid = numericPid(value)
		}
	}
	return nil
}

func daemonPath(current *session, value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Join(current.codexHome, value)
}

func startDaemonThread(current *session, ctx context.Context) error {
	if current.socket == "" {
		return errors.New("daemon_readiness_missing_socket")
	}
	threadID, err := current.threadStarter(ctx, "unix://"+current.socket, current.workspace)
	if err != nil {
		return err
	}
	if threadID == "" {
		return errors.New("app_response_missing_thread_id")
	}
	current.threadID = threadID
	return nil
}

func daemonStarted(current *session) bool {
	for _, command := range current.commands {
		if command == "start" {
			return true
		}
	}
	return false
}

func cleanupDaemon(current *session, ctx context.Context) string {
	_, err := runDaemonCommand(current, ctx, "stop")
	if err != nil {
		return err.Error()
	}
	if current.socket != "" {
		if _, err := os.Stat(current.socket); err == nil {
			return "socket_still_present"
		}
	}
	if result := cleanupProcess(current); result != "stopped" {
		return result
	}
	return "stopped"
}

func cleanupProcess(current *session) string {
	pid := daemonPid(current)
	if pid == 0 {
		return "pid_missing"
	}
	evidence, err := current.readProcess(pid, current.codexHome)
	if err != nil {
		return err.Error()
	}
	if !evidence.Alive {
		return "stopped"
	}
	if !processHasKnownSource(current, evidence) {
		return "process_unowned"
	}
	if !processHasTempHome(current, evidence) {
		return "process_ambiguous"
	}
	if err := current.kill(pid); err != nil {
		return err.Error()
	}
	after, err := current.readProcess(pid, current.codexHome)
	if err != nil {
		return err.Error()
	}
	if after.Alive {
		return "process_still_alive"
	}
	return "stopped"
}

func daemonPid(current *session) int {
	if current.pid != 0 {
		return current.pid
	}
	if current.pidFile == "" {
		return 0
	}
	content, err := os.ReadFile(current.pidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0
	}
	return pid
}

func processHasKnownSource(current *session, evidence ProcessEvidence) bool {
	return processPathAllowed(current, evidence.Command) || processPathAllowed(current, evidence.Executable)
}

func processPathAllowed(current *session, value string) bool {
	path := cleanPath(value)
	return path == current.binary || inside(path, current.codexHome)
}

func processHasTempHome(current *session, evidence ProcessEvidence) bool {
	for _, value := range evidence.Env {
		if value == "CODEX_HOME="+current.codexHome {
			return true
		}
	}
	return false
}

func discoverPidFile(current *session) error {
	if current.pidFile != "" {
		return nil
	}
	return filepath.WalkDir(current.codexHome, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || current.pidFile != "" {
			return err
		}
		if strings.Contains(strings.ToLower(entry.Name()), "pid") {
			current.pidFile = path
		}
		return nil
	})
}

func daemonResult(current *session, started time.Time, cleanup string) DaemonResult {
	return DaemonResult{
		BinaryPath: current.binary,
		Version:    current.version,
		Runtime:    time.Since(started),
		ThreadID:   current.threadID,
		TempClass:  "owned_temp_root",
		Cleanup:    cleanup,
		Commands:   append([]string(nil), current.commands...),
		PathFields: append([]string(nil), current.pathFields...),
	}
}

func newSession(ctx context.Context, options DaemonOptions) (*session, error) {
	binary, err := resolveBinary(options)
	if err != nil {
		return nil, err
	}
	runner := options.Runner
	if runner == nil {
		runner = runCommand
	}
	version, err := verifyVersion(ctx, binary, runner)
	if err != nil {
		return nil, err
	}
	root, err := os.MkdirTemp("", "codex-daemon-readiness-*")
	if err != nil {
		return nil, err
	}
	current := &session{
		binary:        binary,
		version:       version,
		root:          root,
		home:          filepath.Join(root, "home"),
		codexHome:     filepath.Join(root, "codex-home"),
		workspace:     filepath.Join(root, "workspace"),
		runner:        runner,
		threadStarter: options.StartThread,
		readProcess:   options.ReadProcess,
		kill:          options.Kill,
		removeAll:     options.RemoveAll,
	}
	if current.threadStarter == nil {
		current.threadStarter = startThread
	}
	if current.readProcess == nil {
		current.readProcess = processEvidence
	}
	if current.kill == nil {
		current.kill = killProcess
	}
	if current.removeAll == nil {
		current.removeAll = os.RemoveAll
	}
	if err := createDirs(current); err != nil {
		_ = os.RemoveAll(root)
		return nil, err
	}
	return current, nil
}

func resolveBinary(options DaemonOptions) (string, error) {
	candidate := options.BinaryOverride
	if candidate == "" {
		lookup := options.LookupPath
		if lookup == nil {
			lookup = exec.LookPath
		}
		path, err := lookup("codex")
		if err != nil {
			return "", fmt.Errorf("%s: codex executable not found", codePrerequisiteMissing)
		}
		candidate = path
	}
	if !filepath.IsAbs(candidate) {
		return "", fmt.Errorf("%s: codex path must be absolute", codePrerequisiteMissing)
	}
	path, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		path = filepath.Clean(candidate)
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("%s: codex path is not executable", codePrerequisiteMissing)
	}
	return path, nil
}

func verifyVersion(ctx context.Context, binary string, runner DaemonRunner) (string, error) {
	output, err := runner(ctx, DaemonCommand{Name: binary, Args: []string{"--version"}, Env: commandEnvironment(nil)})
	if err != nil {
		return "", fmt.Errorf("%s: %w", codePrerequisiteMissing, err)
	}
	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", fmt.Errorf("%s: empty codex version", codePrerequisiteMissing)
	}
	return version, nil
}

func copyFile(source string, target string) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(target, content, 0o700)
}

func pathFields(values map[string]any) map[string]string {
	fields := map[string]string{}
	for name, value := range values {
		text, ok := value.(string)
		if !ok || text == "" || !namesPath(name) {
			continue
		}
		fields[name] = text
	}
	return fields
}

func namesPath(name string) bool {
	value := strings.ToLower(name)
	for _, token := range []string{"socket", "pidfile", "pid_file", "state", "settings", "log", "binary", "codex_home"} {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}

func namesPid(name string) bool {
	value := strings.ToLower(name)
	return strings.Contains(value, "pid") && !namesPidFile(value)
}

func namesPidFile(name string) bool {
	value := strings.ToLower(name)
	return strings.Contains(value, "pidfile") || strings.Contains(value, "pid_file")
}

func numericPid(value any) int {
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case int:
		return typed
	}
	return 0
}

func inside(path string, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

func startThread(ctx context.Context, endpoint string, workspace string) (string, error) {
	return control.NewUnixClient(endpoint).StartThread(ctx, workspace)
}

func cleanPath(value string) string {
	if value == "" {
		return ""
	}
	path, err := filepath.EvalSymlinks(value)
	if err != nil {
		return filepath.Clean(value)
	}
	return path
}

func runCommand(ctx context.Context, command DaemonCommand) ([]byte, error) {
	item := exec.CommandContext(ctx, command.Name, command.Args...)
	item.Env = command.Env
	return item.CombinedOutput()
}

func commandEnvironment(overrides []string) []string {
	env := minimalEnvironment()
	return append(env, overrides...)
}

func minimalEnvironment() []string {
	env := []string{}
	for _, key := range []string{"PATH", "LANG", "LC_ALL", "SSL_CERT_FILE", "SSL_CERT_DIR"} {
		if value := os.Getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}
	return env
}

func processEvidence(pid int, _ string) (ProcessEvidence, error) {
	root := filepath.Join("/proc", strconv.Itoa(pid))
	command, err := readProcField(filepath.Join(root, "cmdline"))
	if os.IsNotExist(err) {
		return ProcessEvidence{}, nil
	}
	if err != nil {
		return ProcessEvidence{}, err
	}
	executable, err := os.Readlink(filepath.Join(root, "exe"))
	if os.IsNotExist(err) {
		return ProcessEvidence{}, nil
	}
	if err != nil {
		return ProcessEvidence{}, err
	}
	env, err := readProcEnv(filepath.Join(root, "environ"))
	if os.IsNotExist(err) {
		return ProcessEvidence{}, nil
	}
	if err != nil {
		return ProcessEvidence{}, err
	}
	return ProcessEvidence{Alive: true, Command: command, Executable: executable, Env: env}, nil
}

func readProcField(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	fields := bytes.Split(content, []byte{0})
	if len(fields) == 0 {
		return "", nil
	}
	return string(fields[0]), nil
}

func readProcEnv(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := bytes.Split(bytes.TrimRight(content, "\x00"), []byte{0})
	env := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) != 0 {
			env = append(env, string(part))
		}
	}
	return env, nil
}

func killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(syscall.SIGKILL)
}
