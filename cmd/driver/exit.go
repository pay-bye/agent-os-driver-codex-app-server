package main

import (
	"fmt"
	"io"
)

type process struct {
	env    []string
	stdout io.Writer
	stderr io.Writer
}

func run(args []string, env []string, stdout io.Writer, stderr io.Writer) int {
	item := process{
		env:    append([]string(nil), env...),
		stdout: stdout,
		stderr: stderr,
	}
	if err := execute(args, item); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
