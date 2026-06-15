package main

import (
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/install"
)

func loadRecord(home string) (install.Record, error) {
	return install.LoadReadable(home)
}
