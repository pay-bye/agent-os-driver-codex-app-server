package main

import (
	"bytes"
	"testing"
)

func TestRunWritesUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(nil, nil, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	requireContains(t, stderr.String(), "usage: driver")
}
