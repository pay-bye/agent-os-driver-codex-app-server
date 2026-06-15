package control

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
)

func TestEnsureDaemonStartsWhenSocketIsMissing(t *testing.T) {
	var command Command
	home := t.TempDir()
	item := config.Config{
		CodexBin:        "codex",
		CodexHome:       home,
		ControlEndpoint: "unix://" + filepath.Join(home, "missing.sock"),
	}

	err := EnsureDaemon(context.Background(), item, func(_ context.Context, item Command) error {
		command = item
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	requireCommand(t, command, Command{
		Name: "codex",
		Args: []string{"app-server", "daemon", "start"},
		Env:  []string{"CODEX_HOME=" + home},
	})
}

func TestEnsureDaemonRejectsSocketOutsideCodexHome(t *testing.T) {
	home := t.TempDir()
	item := config.Config{
		CodexBin:        "codex",
		CodexHome:       home,
		ControlEndpoint: "unix://" + filepath.Join(t.TempDir(), "missing.sock"),
	}

	err := EnsureDaemon(context.Background(), item, func(context.Context, Command) error {
		t.Fatal("daemon command must not run")
		return nil
	})

	requireError(t, err, "unsafe_codex_home")
}

func TestReadDaemonVersionUsesConfiguredCodexHome(t *testing.T) {
	t.Setenv("CODEX_HOME", "/tmp/ambient-codex-home")
	home := t.TempDir()
	item := config.Config{
		CodexBin:        "codex",
		CodexHome:       home,
		ControlEndpoint: "unix://" + filepath.Join(home, "control.sock"),
	}

	var command Command
	err := ReadDaemonVersion(context.Background(), item, func(_ context.Context, item Command) error {
		command = item
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	requireCommand(t, command, Command{
		Name: "codex",
		Args: []string{"app-server", "daemon", "version"},
		Env:  []string{"CODEX_HOME=" + home},
	})
}
