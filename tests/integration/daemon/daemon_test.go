package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/status"
)

func TestRunRejectsMissingBinary(t *testing.T) {
	_, err := status.CheckDaemon(context.Background(), status.DaemonOptions{LookupPath: func(string) (string, error) {
		return "", errors.New("missing")
	}})

	requireError(t, err, "daemon_readiness_prerequisite_missing")
}

func TestRunRejectsRelativeOverride(t *testing.T) {
	_, err := status.CheckDaemon(context.Background(), status.DaemonOptions{BinaryOverride: "codex"})

	requireError(t, err, "daemon_readiness_prerequisite_missing")
}

func TestRunRecordsCurrentVersion(t *testing.T) {
	bin := fakeBinary(t)
	runner := &fakeRunner{outputs: []string{
		"codex-cli 0.133.0\n",
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 404}),
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 404}),
		daemonJSON(t, map[string]any{"stopped": true}),
	}}
	control := func(context.Context, string, string) (string, error) {
		return "3c9e5a1b7d04f268", nil
	}

	result, err := status.CheckDaemon(context.Background(), status.DaemonOptions{
		BinaryOverride: bin,
		Runner:         runner.run,
		StartThread:    control,
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "codex-cli 0.133.0" {
		t.Fatalf("version = %q, want current version", result.Version)
	}
}

func TestRunUsesTempEnvironmentAndAllowedDaemonCommands(t *testing.T) {
	t.Setenv("CODEX_HOME", "/ambient/codex")
	t.Setenv("AMBIENT_PROBE_ONE", "ambient-one")
	t.Setenv("AMBIENT_PROBE_TWO", "ambient-two")
	bin := fakeBinary(t)
	runner := &fakeRunner{outputs: []string{
		"codex-cli 0.133.0\n",
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 404}),
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 404}),
		daemonJSON(t, map[string]any{"stopped": true}),
	}}
	control := func(context.Context, string, string) (string, error) {
		return "3c9e5a1b7d04f268", nil
	}

	result, err := status.CheckDaemon(context.Background(), status.DaemonOptions{
		BinaryOverride: bin,
		Runner:         runner.run,
		StartThread:    control,
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.ThreadID != "3c9e5a1b7d04f268" {
		t.Fatalf("thread id = %q", result.ThreadID)
	}
	requireArgs(t, runner.commands, [][]string{
		{"--version"},
		{"app-server", "daemon", "start"},
		{"app-server", "daemon", "version"},
		{"app-server", "daemon", "stop"},
	})
	for _, command := range runner.commands[1:] {
		requireTempEnv(t, command.Env)
	}
	if !slices.Contains(result.Commands, "start") ||
		!slices.Contains(result.Commands, "version") ||
		!slices.Contains(result.Commands, "stop") {
		t.Fatalf("commands = %#v", result.Commands)
	}
	for _, command := range runner.commands {
		requireAllowedChildEnv(t, command.Env)
	}
}

func TestRunRejectsResponsePathEscapeAndStopsDaemon(t *testing.T) {
	bin := fakeBinary(t)
	runner := &fakeRunner{outputs: []string{
		"codex-cli 0.133.0\n",
		daemonJSON(t, map[string]any{"socket_path": "/tmp/escaped.sock", "pidfile_path": "app-server.pid"}),
		daemonJSON(t, map[string]any{"stopped": true}),
	}}

	_, err := status.CheckDaemon(context.Background(), status.DaemonOptions{
		BinaryOverride: bin,
		Runner:         runner.run,
	})

	requireError(t, err, "daemon_readiness_path_escape")
	requireArgs(t, runner.commands, [][]string{
		{"--version"},
		{"app-server", "daemon", "start"},
		{"app-server", "daemon", "stop"},
	})
}

func TestRunStopsDaemonAfterProtocolFailure(t *testing.T) {
	bin := fakeBinary(t)
	runner := &fakeRunner{outputs: []string{
		"codex-cli 0.133.0\n",
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 404}),
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 404}),
		daemonJSON(t, map[string]any{"stopped": true}),
	}}
	control := func(context.Context, string, string) (string, error) {
		return "", errors.New("protocol failed")
	}

	_, err := status.CheckDaemon(context.Background(), status.DaemonOptions{
		BinaryOverride: bin,
		Runner:         runner.run,
		StartThread:    control,
	})

	requireError(t, err, "protocol failed")
	requireArgs(t, runner.commands, [][]string{
		{"--version"},
		{"app-server", "daemon", "start"},
		{"app-server", "daemon", "version"},
		{"app-server", "daemon", "stop"},
	})
}

func TestRunKillsOwnedLiveProcessBeforeRemovingTempRoot(t *testing.T) {
	bin := fakeBinary(t)
	runner := &fakeRunner{outputs: []string{
		"codex-cli 0.133.0\n",
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 404}),
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 404}),
		daemonJSON(t, map[string]any{"stopped": true}),
	}}
	reader := &fakeProcessReader{states: []status.ProcessEvidence{
		{Alive: true, Command: bin, Executable: bin},
		{Alive: false},
	}}
	removed := []string{}

	result, err := status.CheckDaemon(context.Background(), status.DaemonOptions{
		BinaryOverride: bin,
		Runner:         runner.run,
		StartThread:    fixedThread,
		ReadProcess:    reader.read,
		Kill:           reader.kill,
		RemoveAll: func(path string) error {
			removed = append(removed, path)
			return nil
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Cleanup != "stopped" {
		t.Fatalf("cleanup = %q, want stopped", result.Cleanup)
	}
	if !slices.Equal(reader.killed, []int{404}) {
		t.Fatalf("killed = %#v, want owned pid", reader.killed)
	}
	if len(removed) != 1 {
		t.Fatalf("removed roots = %#v, want one removal", removed)
	}
}

func TestRunPreservesTempRootForUnownedLiveProcess(t *testing.T) {
	bin := fakeBinary(t)
	runner := &fakeRunner{outputs: []string{
		"codex-cli 0.133.0\n",
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 405}),
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 405}),
		daemonJSON(t, map[string]any{"stopped": true}),
	}}
	reader := &fakeProcessReader{states: []status.ProcessEvidence{
		{Alive: true, Command: "/usr/bin/other", Executable: "/usr/bin/other", Env: []string{"CODEX_HOME=/tmp/other"}},
	}}
	removed := []string{}

	result, err := status.CheckDaemon(context.Background(), status.DaemonOptions{
		BinaryOverride: bin,
		Runner:         runner.run,
		StartThread:    fixedThread,
		ReadProcess:    reader.read,
		Kill:           reader.kill,
		RemoveAll: func(path string) error {
			removed = append(removed, path)
			return nil
		},
	})

	requireError(t, err, "daemon_readiness_cleanup_failed")
	if result.Cleanup != "process_unowned" {
		t.Fatalf("cleanup = %q, want process_unowned", result.Cleanup)
	}
	if len(reader.killed) != 0 {
		t.Fatalf("killed = %#v, want no unowned kill", reader.killed)
	}
	if len(removed) != 0 {
		t.Fatalf("removed roots = %#v, want preserved temp root", removed)
	}
}

func TestRunPreservesTempRootForAmbiguousLiveProcess(t *testing.T) {
	bin := fakeBinary(t)
	runner := &fakeRunner{outputs: []string{
		"codex-cli 0.133.0\n",
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 406}),
		daemonJSON(t, map[string]any{"socket_path": "socket", "pidfile_path": "app-server.pid", "pid": 406}),
		daemonJSON(t, map[string]any{"stopped": true}),
	}}
	reader := &fakeProcessReader{states: []status.ProcessEvidence{
		{Alive: true, Command: bin, Executable: bin, Env: []string{"OTHER=1"}},
	}}
	removed := []string{}

	result, err := status.CheckDaemon(context.Background(), status.DaemonOptions{
		BinaryOverride: bin,
		Runner:         runner.run,
		StartThread:    fixedThread,
		ReadProcess:    reader.read,
		Kill:           reader.kill,
		RemoveAll: func(path string) error {
			removed = append(removed, path)
			return nil
		},
	})

	requireError(t, err, "daemon_readiness_cleanup_failed")
	if result.Cleanup != "process_ambiguous" {
		t.Fatalf("cleanup = %q, want process_ambiguous", result.Cleanup)
	}
	if len(reader.killed) != 0 {
		t.Fatalf("killed = %#v, want no ambiguous kill", reader.killed)
	}
	if len(removed) != 0 {
		t.Fatalf("removed roots = %#v, want preserved temp root", removed)
	}
}

func TestLiveDaemonReadiness(t *testing.T) {
	if os.Getenv("RUN_DAEMON_READINESS") != "1" {
		t.Skip("daemon readiness is non-default")
	}
	result, err := status.CheckDaemon(context.Background(), status.DaemonOptions{
		BinaryOverride: os.Getenv("CODEX_READINESS_BIN"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("binary=%s version=%s runtime=%s thread=%s commands=%s",
		result.BinaryPath,
		result.Version,
		result.Runtime,
		result.ThreadID,
		strings.Join(result.Commands, ","),
	)
	t.Logf("temp=%s cleanup=%s paths=%s",
		result.TempClass,
		result.Cleanup,
		strings.Join(result.PathFields, ","),
	)
}

type fakeRunner struct {
	commands []status.DaemonCommand
	outputs  []string
}

func (r *fakeRunner) run(_ context.Context, command status.DaemonCommand) ([]byte, error) {
	r.commands = append(r.commands, command)
	if len(r.outputs) == 0 {
		return nil, errors.New("unexpected command")
	}
	output := r.outputs[0]
	r.outputs = r.outputs[1:]
	return []byte(output), nil
}

type fakeProcessReader struct {
	killed []int
	states []status.ProcessEvidence
}

func (p *fakeProcessReader) read(pid int, codexHome string) (status.ProcessEvidence, error) {
	if len(p.states) == 0 {
		return status.ProcessEvidence{}, errors.New("unexpected process read")
	}
	state := p.states[0]
	p.states = p.states[1:]
	if state.Alive && len(state.Env) == 0 {
		state.Env = []string{"CODEX_HOME=" + codexHome}
	}
	return state, nil
}

func (p *fakeProcessReader) kill(pid int) error {
	p.killed = append(p.killed, pid)
	return nil
}

func fixedThread(context.Context, string, string) (string, error) {
	return "3c9e5a1b7d04f268", nil
}

func fakeBinary(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "codex")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func daemonJSON(t *testing.T, values map[string]any) string {
	t.Helper()

	content, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	return string(content) + "\n"
}

func requireError(t *testing.T, err error, code string) {
	t.Helper()

	if err == nil || !strings.Contains(err.Error(), code) {
		t.Fatalf("error = %v, want %s", err, code)
	}
}

func requireArgs(t *testing.T, commands []status.DaemonCommand, want [][]string) {
	t.Helper()

	if len(commands) != len(want) {
		t.Fatalf("commands = %d, want %d", len(commands), len(want))
	}
	for index, command := range commands {
		if !slices.Equal(command.Args, want[index]) {
			t.Fatalf("command[%d] args = %#v, want %#v", index, command.Args, want[index])
		}
	}
}

func requireTempEnv(t *testing.T, env []string) {
	t.Helper()

	values := map[string]string{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key] = value
		}
	}
	codexHome := values["CODEX_HOME"]
	if codexHome == "" {
		t.Fatal("CODEX_HOME missing")
	}
	for _, key := range []string{"HOME", "TMPDIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR"} {
		if !strings.HasPrefix(values[key], filepath.Dir(codexHome)) {
			t.Fatalf("%s = %q, want temp root sibling of CODEX_HOME %q", key, values[key], codexHome)
		}
	}
}

func requireAllowedChildEnv(t *testing.T, env []string) {
	t.Helper()

	for _, value := range env {
		key, _, ok := strings.Cut(value, "=")
		if !ok {
			t.Fatalf("env item = %q, want KEY=value", value)
		}
		if !allowedChildEnvKey(key) {
			t.Fatalf("env key = %q, want closed child environment", key)
		}
	}
}

func allowedChildEnvKey(key string) bool {
	return slices.Contains([]string{
		"PATH",
		"LANG",
		"LC_ALL",
		"SSL_CERT_FILE",
		"SSL_CERT_DIR",
		"HOME",
		"CODEX_HOME",
		"TMPDIR",
		"XDG_CONFIG_HOME",
		"XDG_CACHE_HOME",
		"XDG_RUNTIME_DIR",
	}, key)
}
