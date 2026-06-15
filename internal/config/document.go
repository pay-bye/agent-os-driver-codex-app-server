package config

import (
	"fmt"
)

type Need struct {
	Kind    string         `json:"need_kind"`
	Payload map[string]any `json:"payload,omitempty"`
}

type Config struct {
	InvocationBaseURL string `json:"invocation_base_url"`
	ChannelKey        string `json:"channel_key"`
	LeaseSeconds      int    `json:"lease_seconds"`
	CodexBin          string `json:"codex_bin"`
	CodexHome         string `json:"codex_home"`
	ControlEndpoint   string `json:"control_endpoint"`
	WorkspaceRoot     string `json:"workspace_root"`
	InputTextPointer  string `json:"input_text_pointer"`
	CompletionNeeds   []Need `json:"completion_needs"`
	FailureNeeds      []Need `json:"failure_needs"`
	RedactionMode     string `json:"redaction_mode"`
}

func (c Config) Validate(check CodexCheck) error {
	if err := c.validateFields(); err != nil {
		return err
	}
	if check != nil {
		if err := check(c.CodexBin); err != nil {
			return fmt.Errorf("codex_unavailable: %w", err)
		}
	}
	return nil
}

func (c Config) validateFields() error {
	checks := []func() error{
		func() error { return requireInvocationURL(c.InvocationBaseURL) },
		func() error { return requireNonEmpty("invalid_channel_key", c.ChannelKey) },
		func() error { return requirePositiveLease(c.LeaseSeconds) },
		func() error { return requireNonEmpty("invalid_codex_bin", c.CodexBin) },
		func() error { return requireAbsolutePath("invalid_codex_home", c.CodexHome) },
		func() error { return requireControlEndpoint(c.ControlEndpoint, c.CodexHome) },
		func() error { return requireAbsolutePath("invalid_workspace_root", c.WorkspaceRoot) },
		func() error { return requirePointer(c.InputTextPointer) },
		func() error { return requireNeeds(c.CompletionNeeds) },
		func() error { return requireNeeds(c.FailureNeeds) },
		func() error { return requireRedactionMode(c.RedactionMode) },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}
