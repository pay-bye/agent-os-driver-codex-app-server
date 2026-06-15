package compatibility

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type CommandRunner func(context.Context, string, ...string) ([]byte, error)

type CommandAppMetadata struct {
	CodexBin string
	Env      []string
	Run      CommandRunner
}

func (p CommandAppMetadata) Metadata(ctx context.Context) (AppMetadata, error) {
	if p.CodexBin == "" {
		return AppMetadata{}, ErrCodexMissing
	}
	version, err := p.version(ctx)
	if err != nil {
		return AppMetadata{}, err
	}
	root, err := os.MkdirTemp("", "codex-app-server-schema-*")
	if err != nil {
		return AppMetadata{}, err
	}
	defer os.RemoveAll(root)

	if err := p.run(ctx, root); err != nil {
		return AppMetadata{}, fmt.Errorf("%w: %v", ErrAppProtocolDrift, err)
	}
	if err := requireProtocolFiles(root); err != nil {
		return AppMetadata{}, err
	}
	digest, err := schemaDigest(root)
	if err != nil {
		return AppMetadata{}, err
	}
	return AppMetadata{
		CodexVersion:   version,
		SchemaDigest:   digest,
		SchemaFiles:    requiredProtocolFiles(),
		Methods:        []string{"initialize", "thread/start", "turn/start", "turn/interrupt"},
		Notifications:  []string{"turn/completed"},
		ControlSurface: "uds_websocket",
	}, nil
}

func (p CommandAppMetadata) version(ctx context.Context) (string, error) {
	output, err := p.runCommand(ctx, "--version")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCodexMissing, err)
	}
	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", ErrCodexMissing
	}
	return version, nil
}

func (p CommandAppMetadata) runCommand(ctx context.Context, args ...string) ([]byte, error) {
	if p.Run != nil {
		return p.Run(ctx, p.CodexBin, args...)
	}
	command := exec.CommandContext(ctx, p.CodexBin, args...)
	command.Env = p.Env
	return command.CombinedOutput()
}

func (p CommandAppMetadata) run(ctx context.Context, root string) error {
	_, err := p.runCommand(ctx, "app-server", "generate-json-schema", "--experimental", "--out", root)
	return err
}

type hashWriter interface {
	Write([]byte) (int, error)
}

func requireProtocolFiles(root string) error {
	for _, path := range requiredProtocolFiles() {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			return ErrAppProtocolDrift
		}
	}
	return nil
}

func requiredProtocolFiles() []string {
	return []string{
		"codex_app_server_protocol.schemas.json",
		"codex_app_server_protocol.v2.schemas.json",
		"v2/ThreadStartParams.json",
		"v2/ThreadStartResponse.json",
		"v2/TurnStartParams.json",
		"v2/TurnStartResponse.json",
		"v2/TurnCompletedNotification.json",
		"v2/TurnInterruptParams.json",
		"v2/TurnInterruptResponse.json",
		"v2/RemoteControlEnableResponse.json",
		"v2/RemoteControlDisableResponse.json",
		"v2/RemoteControlStatusReadResponse.json",
		"v2/RemoteControlStatusChangedNotification.json",
	}
}

func schemaDigest(root string) (string, error) {
	paths, err := filesUnder(root)
	if err != nil {
		return "", err
	}
	slices.Sort(paths)
	hash := sha256.New()
	for _, path := range paths {
		if err := hashSchemaFile(hash, root, path); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

func hashSchemaFile(hash hashWriter, root string, path string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	content, err := canonicalContent(path)
	if err != nil {
		return err
	}
	hash.Write([]byte(filepath.ToSlash(relative)))
	hash.Write([]byte{0})
	hash.Write(content)
	hash.Write([]byte{0})
	return nil
}

func canonicalContent(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if filepath.Ext(path) != ".json" {
		return content, nil
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, ErrAppProtocolDrift
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

func filesUnder(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}
