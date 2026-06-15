package compatibility

import (
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/invoke"
)

type Route = invoke.Route

type Metadata = invoke.Metadata

type AppMetadata struct {
	CodexVersion   string
	SchemaDigest   string
	SchemaFiles    []string
	Methods        []string
	Notifications  []string
	ControlSurface string
}
