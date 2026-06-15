package control

import (
	"context"
	"errors"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CommandRunner func(context.Context, Command) error

func NewCommandRunner(env []string) CommandRunner {
	return func(ctx context.Context, item Command) error {
		return runCommand(ctx, item, env)
	}
}

type Command struct {
	Name string
	Args []string
	Env  []string
}

func EnsureDaemon(ctx context.Context, item config.Config, runner CommandRunner) error {
	if NewUnixClient(item.ControlEndpoint).Ready(ctx) {
		return nil
	}
	return runDaemonCommand(ctx, item, "start", runner)
}

func ReadDaemonVersion(ctx context.Context, item config.Config, runner CommandRunner) error {
	return runDaemonCommand(ctx, item, "version", runner)
}

func runDaemonCommand(ctx context.Context, item config.Config, action string, runner CommandRunner) error {
	command, err := daemonCommand(item, action)
	if err != nil {
		return err
	}
	if runner == nil {
		runner = NewCommandRunner(os.Environ())
	}
	return runner(ctx, command)
}

func daemonCommand(item config.Config, action string) (Command, error) {
	if !safeHomeOwnsSocket(item) {
		return Command{}, errors.New("unsafe_codex_home")
	}
	return Command{
		Name: item.CodexBin,
		Args: []string{"app-server", "daemon", action},
		Env:  []string{"CODEX_HOME=" + filepath.Clean(item.CodexHome)},
	}, nil
}

func safeHomeOwnsSocket(item config.Config) bool {
	return config.SocketInsideHome(item.ControlEndpoint, item.CodexHome)
}

func runCommand(ctx context.Context, item Command, env []string) error {
	command := exec.CommandContext(ctx, item.Name, item.Args...)
	command.Env = commandEnvironment(env, item.Env)
	return command.Run()
}

func commandEnvironment(base []string, overrides []string) []string {
	env := make([]string, 0, len(base)+len(overrides))
	for _, value := range base {
		if strings.HasPrefix(value, "CODEX_HOME=") {
			continue
		}
		env = append(env, value)
	}
	return append(env, overrides...)
}
