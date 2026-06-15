package install

import (
	_ "embed"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
	"os/exec"
)

func checkExecutable(path string) error {
	return exec.Command(path, "app-server", "--help").Run()
}

func codexCheck(check config.CodexCheck) config.CodexCheck {
	if check != nil {
		return check
	}
	return checkExecutable
}
