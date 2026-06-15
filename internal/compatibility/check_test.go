package compatibility

import (
	"context"
	"errors"
	"testing"
)

func TestVerifierClassifiesEndpointMismatches(t *testing.T) {
	requireCompatibilityMismatches(t, []mismatchCase{
		{
			name:      "endpoint unavailable",
			invokeErr: ErrEndpointUnavailable,
			want:      ErrEndpointUnavailable,
			code:      "endpoint_unavailable",
		},
		{
			name:      "metadata malformed",
			invokeErr: ErrMetadataMalformed,
			want:      ErrMetadataMalformed,
			code:      "metadata_malformed",
		},
	})
}

func TestVerifierClassifiesContractVersionMismatches(t *testing.T) {
	requireCompatibilityMismatches(t, []mismatchCase{
		{
			name:       "contract too old",
			invocation: metadataWithVersion("v0"),
			want:       ErrContractTooOld,
			code:       "contract_too_old",
		},
		{
			name:       "contract too new",
			invocation: metadataWithVersion("v2"),
			want:       ErrContractTooNew,
			code:       "contract_too_new",
		},
	})
}

func TestVerifierClassifiesInvocationPayloadMismatches(t *testing.T) {
	requireCompatibilityMismatches(t, []mismatchCase{
		{
			name:       "schema digest",
			invocation: metadataWithDigest(otherDigest),
			want:       ErrSchemaDigestUnaccepted,
			code:       "schema_digest_unaccepted",
		},
		{
			name:       "feature missing",
			invocation: metadataWithoutFeature("lease_ack"),
			want:       ErrFeatureMissing,
			code:       "feature_missing",
		},
		{
			name:       "route inventory malformed",
			invocation: metadataWithoutRoute(Route{Method: "POST", Path: "/ack"}),
			want:       ErrMetadataMalformed,
			code:       "metadata_malformed",
		},
	})
}

func TestVerifierClassifiesAppProtocolMismatches(t *testing.T) {
	requireCompatibilityMismatches(t, []mismatchCase{
		{
			name: "app method missing",
			app:  appWithoutMethod("thread/start"),
			want: ErrAppProtocolDrift,
			code: "app_protocol_drift",
		},
	})
}

type mismatchCase struct {
	name       string
	invocation Metadata
	invokeErr  error
	app        AppMetadata
	appErr     error
	want       error
	code       string
}

func requireCompatibilityMismatches(t *testing.T, cases []mismatchCase) {
	t.Helper()

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result, err := verifyMismatch(test)

			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if result.DiagnosticCode != test.code {
				t.Fatalf("code = %q, want %q", result.DiagnosticCode, test.code)
			}
		})
	}
}

func verifyMismatch(test mismatchCase) (Result, error) {
	return Verifier{
		Requirements: requirementsFixture(),
		Invocation:   invocationReader{metadata: chooseMetadata(test.invocation), err: test.invokeErr},
		App:          appReader{metadata: chooseApp(test), err: test.appErr},
	}.Verify(context.Background())
}

func TestVerifierReturnsObservedCompatibilityEvidence(t *testing.T) {
	result, err := Verifier{
		Requirements: requirementsFixture(),
		Invocation:   invocationReader{metadata: fullMetadata()},
		App:          appReader{metadata: fullAppMetadata()},
	}.Verify(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.DiagnosticCode != "compatible" {
		t.Fatalf("code = %q, want compatible", result.DiagnosticCode)
	}
	if result.InvocationMetadata.SchemaSetDigest != acceptedDigest {
		t.Fatalf("digest = %q, want %q", result.InvocationMetadata.SchemaSetDigest, acceptedDigest)
	}
	if result.AppServerDigest != observedAppDigest {
		t.Fatalf("app digest = %q, want %q", result.AppServerDigest, observedAppDigest)
	}
}

func TestVerifierAcceptsCurrentAppVersionAndRecordsSchemaDigest(t *testing.T) {
	app := fullAppMetadata()
	app.CodexVersion = "codex-cli 0.133.0"
	app.SchemaDigest = otherDigest

	result, err := Verifier{
		Requirements: requirementsFixture(),
		Invocation:   invocationReader{metadata: fullMetadata()},
		App:          appReader{metadata: app},
	}.Verify(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if result.AppServerDigest != otherDigest {
		t.Fatalf("app digest = %q, want %q", result.AppServerDigest, otherDigest)
	}
}

type invocationReader struct {
	metadata Metadata
	err      error
}

func (p invocationReader) Metadata(context.Context) (Metadata, error) {
	return p.metadata, p.err
}

type appReader struct {
	metadata AppMetadata
	err      error
}

func (p appReader) Metadata(context.Context) (AppMetadata, error) {
	return p.metadata, p.err
}

func chooseMetadata(metadata Metadata) Metadata {
	if metadata.ContractVersion == "" {
		return fullMetadata()
	}
	return metadata
}

func chooseApp(test mismatchCase) AppMetadata {
	if test.app.SchemaDigest != "" || test.appErr != nil {
		return test.app
	}
	return fullAppMetadata()
}

func metadataWithVersion(value string) Metadata {
	metadata := fullMetadata()
	metadata.ContractVersion = value
	return metadata
}

func metadataWithDigest(value string) Metadata {
	metadata := fullMetadata()
	metadata.SchemaSetDigest = value
	return metadata
}

func metadataWithoutFeature(value string) Metadata {
	metadata := fullMetadata()
	metadata.Features = without(metadata.Features, value)
	return metadata
}

func metadataWithoutRoute(value Route) Metadata {
	metadata := fullMetadata()
	for index, route := range metadata.Routes {
		if route == value {
			metadata.Routes = append(metadata.Routes[:index], metadata.Routes[index+1:]...)
			return metadata
		}
	}
	return metadata
}

func without(values []string, rejected string) []string {
	var kept []string
	for _, value := range values {
		if value != rejected {
			kept = append(kept, value)
		}
	}
	return kept
}

func fullMetadata() Metadata {
	return Metadata{
		ContractVersion: "v1",
		SchemaSetDigest: acceptedDigest,
		Features: []string{
			"lease_claim",
			"lease_extend",
			"lease_ack",
			"lease_nack",
			"lease_capability",
			"declared_needs",
			"failure_payload",
		},
		Routes: []Route{
			{Method: "POST", Path: "/claim"},
			{Method: "POST", Path: "/ack"},
			{Method: "POST", Path: "/nack"},
			{Method: "POST", Path: "/extend"},
			{Method: "GET", Path: "/compatibility"},
		},
	}
}

func fullAppMetadata() AppMetadata {
	return AppMetadata{
		CodexVersion:   currentVersion,
		SchemaDigest:   observedAppDigest,
		SchemaFiles:    requiredSchemaFiles(),
		Methods:        []string{"initialize", "thread/start", "turn/start", "turn/interrupt"},
		Notifications:  []string{"turn/completed"},
		ControlSurface: "uds_websocket",
	}
}

func appWithDigest(value string) AppMetadata {
	metadata := fullAppMetadata()
	metadata.SchemaDigest = value
	return metadata
}

func appWithoutMethod(value string) AppMetadata {
	metadata := fullAppMetadata()
	metadata.Methods = without(metadata.Methods, value)
	return metadata
}

func requirementsFixture() Requirements {
	return Requirements{
		AcceptedVersions:         []string{"v1"},
		AcceptedSchemaSetDigests: []string{acceptedDigest},
		RequiredFeatures: []string{
			"lease_claim",
			"lease_extend",
			"lease_ack",
			"lease_nack",
			"lease_capability",
			"declared_needs",
			"failure_payload",
		},
		RequiredRoutes: []Route{
			{Method: "POST", Path: "/claim"},
			{Method: "POST", Path: "/ack"},
			{Method: "POST", Path: "/nack"},
			{Method: "POST", Path: "/extend"},
			{Method: "GET", Path: "/compatibility"},
		},
		InstallRecord: RecordRequirement{
			CurrentVersion:         3,
			MinimumReadableVersion: 1,
		},
		CodexCLI: CLIRequirement{Version: VersionRequirement{Evidence: "record_current"}},
		AppServer: AppRequirement{
			SchemaDigestCanonicalization: "json_sort_keys_v1",
			RequiredSchemaFiles:          requiredSchemaFiles(),
			RequiredMethods:              []string{"initialize", "thread/start", "turn/start", "turn/interrupt"},
			RequiredNotifications:        []string{"turn/completed"},
			ControlSurface:               "uds_websocket",
		},
	}
}

const acceptedDigest = "sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac"
const otherDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const observedAppDigest = "sha256:588e3e4ad47defffca22dba7769d0c902756165931eedee139e6e4f9efdac16d"
const currentVersion = "codex-cli test-current"
