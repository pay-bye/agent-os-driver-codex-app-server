package config

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

func requireInvocationURL(value string) error {
	item, err := url.Parse(value)
	if err != nil || item.Scheme == "" || item.Host == "" {
		return errors.New("invalid_invocation_base_url")
	}
	if item.Scheme != "http" && item.Scheme != "https" {
		return errors.New("invalid_invocation_base_url")
	}
	if item.User != nil {
		return errors.New("invalid_invocation_base_url")
	}
	return nil
}

func requireNonEmpty(code string, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(code)
	}
	return nil
}

func requirePositiveLease(value int) error {
	if value <= 0 {
		return errors.New("invalid_lease_seconds")
	}
	return nil
}

func requireControlEndpoint(value string, home string) error {
	if !SocketInsideHome(value, home) {
		return errors.New("invalid_control_endpoint")
	}
	return nil
}

func requireAbsolutePath(code string, value string) error {
	if !filepath.IsAbs(value) {
		return errors.New(code)
	}
	return nil
}

func requirePointer(value string) error {
	if value == "" || !strings.HasPrefix(value, "/") {
		return errors.New("invalid_input_text_pointer")
	}
	return nil
}

func requireNeeds(items []Need) error {
	for _, item := range items {
		if strings.TrimSpace(item.Kind) == "" {
			return errors.New("invalid_declared_need")
		}
	}
	return nil
}

func requireRedactionMode(value string) error {
	if value != "metadata_only" {
		return errors.New("invalid_redaction_mode")
	}
	return nil
}
