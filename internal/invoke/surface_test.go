package invoke

import (
	"errors"
	"testing"
)

func TestMetadataRejectsNonInventoryRoutes(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
	}{
		{name: "missing compatibility route", metadata: metadataWithoutRoute(Route{Method: "GET", Path: "/compatibility"})},
		{name: "unknown route", metadata: metadataWithRoute(Route{Method: "POST", Path: "/unknown"})},
		{name: "duplicate route", metadata: metadataWithRoute(Route{Method: "POST", Path: "/claim"})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.metadata.Validate()

			if !errors.Is(err, ErrMetadataMalformed) {
				t.Fatalf("error = %v, want metadata_malformed", err)
			}
		})
	}
}

func metadataWithoutRoute(value Route) Metadata {
	metadata := metadataFixture()
	for index, route := range metadata.Routes {
		if route == value {
			metadata.Routes = append(metadata.Routes[:index], metadata.Routes[index+1:]...)
			return metadata
		}
	}
	return metadata
}

func metadataWithRoute(value Route) Metadata {
	metadata := metadataFixture()
	metadata.Routes = append(metadata.Routes, value)
	return metadata
}

func metadataFixture() Metadata {
	return Metadata{
		ContractVersion: "v1",
		SchemaSetDigest: "sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac",
		Features: []string{
			"lease_claim",
			"lease_extend",
			"lease_ack",
			"lease_nack",
			"lease_capability",
			"declared_needs",
			"failure_payload",
		},
		Routes: PublishedRoutes(),
	}
}
