package compatibility

type InstallEvidence struct {
	Requirements       Requirements `json:"requirements"`
	InvocationMetadata Metadata     `json:"observed_invocation_metadata"`
	AppServerDigest    string       `json:"observed_app_server_digest"`
}

type Result struct {
	Requirements       Requirements
	InvocationMetadata Metadata
	AppServerDigest    string
	DiagnosticCode     string
}
