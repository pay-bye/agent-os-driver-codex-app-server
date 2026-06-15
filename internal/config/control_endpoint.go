package config

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

func ControlSocketPath(endpoint string) (string, error) {
	item, err := url.Parse(endpoint)
	if err != nil || item.Scheme != "unix" {
		return "", errors.New("invalid_control_endpoint")
	}
	if item.Path == "" || !filepath.IsAbs(item.Path) {
		return "", errors.New("invalid_control_endpoint")
	}
	return item.Path, nil
}

func SocketInsideHome(endpoint string, home string) bool {
	path, err := ControlSocketPath(endpoint)
	if err != nil {
		return false
	}
	return pathInside(home, path)
}

func pathInside(root string, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
