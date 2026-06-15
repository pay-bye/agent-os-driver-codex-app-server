package install

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"io"
	"os"
)

//go:embed driver_manifest.json
var defaultManifest []byte

type manifestFile struct {
	SchemaVersion int                        `json:"schema_version"`
	Compatibility compatibility.Requirements `json:"compatibility"`
}

func DefaultRequirements() (compatibility.Requirements, error) {
	return decode(bytes.NewReader(defaultManifest))
}

func ReadRequirements(path string) (compatibility.Requirements, error) {
	file, err := os.Open(path)
	if err != nil {
		return compatibility.Requirements{}, err
	}
	defer file.Close()

	return decode(file)
}

func decode(source io.Reader) (compatibility.Requirements, error) {
	var item manifestFile
	decoder := json.NewDecoder(source)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&item); err != nil {
		return compatibility.Requirements{}, malformed(err)
	}
	if item.SchemaVersion != 1 {
		return compatibility.Requirements{}, malformed(fmt.Errorf("schema_version=%d", item.SchemaVersion))
	}
	if err := item.Compatibility.Validate(); err != nil {
		return compatibility.Requirements{}, err
	}
	return item.Compatibility, nil
}

func malformed(err error) error {
	if err == nil {
		return compatibility.ErrMetadataMalformed
	}
	return fmt.Errorf("%w: %v", compatibility.ErrMetadataMalformed, err)
}
