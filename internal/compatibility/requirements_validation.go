package compatibility

import (
	"fmt"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/invoke"
	"regexp"
	"strings"
)

var (
	versionPattern = regexp.MustCompile(`^v[0-9]+$`)
	digestPattern  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

func requireVersions(values []string) error {
	if len(values) == 0 {
		return malformed(nil)
	}
	for _, value := range values {
		if !versionPattern.MatchString(value) {
			return malformed(fmt.Errorf("version=%q", value))
		}
	}
	return nil
}

func requireDigests(values []string) error {
	if len(values) == 0 {
		return malformed(nil)
	}
	for _, value := range values {
		if err := requireDigest(value); err != nil {
			return err
		}
	}
	return nil
}

func requireDigest(value string) error {
	if !digestPattern.MatchString(value) {
		return malformed(fmt.Errorf("digest=%q", value))
	}
	return nil
}

func requireFeatures(values []string) error {
	if len(values) == 0 {
		return malformed(nil)
	}
	for _, value := range values {
		if !acceptedFeature(value) {
			return malformed(fmt.Errorf("feature=%q", value))
		}
	}
	return nil
}

func requireRoutes(values []Route) error {
	return invoke.ValidateRoutes(values)
}

func requireRecord(value RecordRequirement) error {
	if value.CurrentVersion < 1 || value.MinimumReadableVersion < 1 {
		return malformed(nil)
	}
	if value.MinimumReadableVersion > value.CurrentVersion {
		return malformed(nil)
	}
	return nil
}

func requireCLI(value CLIRequirement) error {
	if value.Version.Evidence != "record_current" {
		return malformed(fmt.Errorf("codex_cli.version.evidence=%q", value.Version.Evidence))
	}
	return nil
}

func requireApp(value AppRequirement) error {
	if value.SchemaDigestCanonicalization != "json_sort_keys_v1" {
		return malformed(fmt.Errorf("schema_digest_canonicalization=%q", value.SchemaDigestCanonicalization))
	}
	if err := requireSchemaFiles(value.RequiredSchemaFiles); err != nil {
		return err
	}
	if err := requireNonEmpty("method", value.RequiredMethods); err != nil {
		return err
	}
	if err := requireNonEmpty("notification", value.RequiredNotifications); err != nil {
		return err
	}
	if value.ControlSurface != "uds_websocket" {
		return malformed(fmt.Errorf("control_surface=%q", value.ControlSurface))
	}
	return nil
}

func requireNonEmpty(name string, values []string) error {
	if len(values) == 0 {
		return malformed(fmt.Errorf("%s missing", name))
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return malformed(fmt.Errorf("%s empty", name))
		}
	}
	return nil
}

func requireSchemaFiles(values []string) error {
	if len(values) == 0 {
		return malformed(fmt.Errorf("schema files missing"))
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") {
			return malformed(fmt.Errorf("schema file=%q", value))
		}
	}
	return nil
}

func acceptedFeature(value string) bool {
	switch value {
	case "lease_claim", "lease_extend", "lease_ack", "lease_nack", "lease_capability", "declared_needs", "failure_payload":
		return true
	default:
		return false
	}
}
