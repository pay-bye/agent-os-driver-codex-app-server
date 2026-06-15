package main

import (
	"fmt"
	"strings"
)

func execute(args []string, item process) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "install":
		return installCommand(args[1:], item)
	case "upgrade":
		return upgradeCommand(args[1:], item)
	case "remove":
		return removeCommand(args[1:])
	case "run":
		return runCommand(args[1:], item)
	case "status":
		return statusCommand(args[1:], item)
	case "doctor":
		return doctorCommand(args[1:], item)
	default:
		return usage()
	}
}

func usage() error {
	commands := []string{
		"install <config.json> <home>",
		"upgrade <home>",
		"run <home>",
		"remove <home>",
		"status <home>",
		"doctor <home>",
	}
	return fmt.Errorf("usage: driver %s", strings.Join(commands, " | driver "))
}
