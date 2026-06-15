package compatibility

import (
	"errors"
	"fmt"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/invoke"
)

var (
	ErrEndpointUnavailable          = invoke.ErrEndpointUnavailable
	ErrMetadataMalformed            = invoke.ErrMetadataMalformed
	ErrContractTooOld               = errors.New("contract_too_old")
	ErrContractTooNew               = errors.New("contract_too_new")
	ErrSchemaDigestUnaccepted       = errors.New("schema_digest_unaccepted")
	ErrFeatureMissing               = errors.New("feature_missing")
	ErrRouteMissing                 = errors.New("route_missing")
	ErrAppProtocolDrift             = errors.New("app_protocol_drift")
	ErrCodexMissing                 = errors.New("codex_missing")
	ErrCodexVersionUnsupported      = errors.New("codex_version_unsupported")
	ErrInstallRecordUpgradeRequired = errors.New("install_record_upgrade_required")
)

func Code(err error) string {
	switch {
	case err == nil:
		return "compatible"
	case errors.Is(err, ErrEndpointUnavailable):
		return "endpoint_unavailable"
	case errors.Is(err, ErrMetadataMalformed):
		return "metadata_malformed"
	case errors.Is(err, ErrContractTooOld):
		return "contract_too_old"
	case errors.Is(err, ErrContractTooNew):
		return "contract_too_new"
	case errors.Is(err, ErrSchemaDigestUnaccepted):
		return "schema_digest_unaccepted"
	case errors.Is(err, ErrFeatureMissing):
		return "feature_missing"
	case errors.Is(err, ErrRouteMissing):
		return "route_missing"
	case errors.Is(err, ErrAppProtocolDrift):
		return "app_protocol_drift"
	case errors.Is(err, ErrCodexMissing):
		return "codex_missing"
	case errors.Is(err, ErrCodexVersionUnsupported):
		return "codex_version_unsupported"
	case errors.Is(err, ErrInstallRecordUpgradeRequired):
		return "install_record_upgrade_required"
	default:
		return "metadata_malformed"
	}
}

func malformed(err error) error {
	if err == nil {
		return ErrMetadataMalformed
	}
	return fmt.Errorf("%w: %v", ErrMetadataMalformed, err)
}
