package invoke

import (
	"context"
	"errors"
	"testing"
)

func TestClientReadsCompatibilityMetadata(t *testing.T) {
	server := newHTTPServer(t)
	defer server.Close()
	client := New(server.URL, server.Client())

	metadata, err := client.Metadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if metadata.ContractVersion != "v1" {
		t.Fatalf("contract version = %q, want v1", metadata.ContractVersion)
	}
	requireRoutes(t, server.Routes(), []string{"/compatibility"})
}

func TestClientClassifiesMalformedCompatibilityMetadata(t *testing.T) {
	server := newHTTPServer(t)
	server.malformedMetadata = true
	defer server.Close()
	client := New(server.URL, server.Client())

	_, err := client.Metadata(context.Background())

	if !errors.Is(err, ErrMetadataMalformed) {
		t.Fatalf("error = %v, want metadata_malformed", err)
	}
}

func TestClientRejectsCompatibilityMetadataWithoutPublishedRoute(t *testing.T) {
	server := newHTTPServer(t)
	server.omitCompatibilityRoute = true
	defer server.Close()
	client := New(server.URL, server.Client())

	_, err := client.Metadata(context.Background())

	if !errors.Is(err, ErrMetadataMalformed) {
		t.Fatalf("error = %v, want metadata_malformed", err)
	}
}

func TestClientRejectsCompatibilityMetadataWithUnknownRoute(t *testing.T) {
	server := newHTTPServer(t)
	server.unknownMetadataRoute = true
	defer server.Close()
	client := New(server.URL, server.Client())

	_, err := client.Metadata(context.Background())

	if !errors.Is(err, ErrMetadataMalformed) {
		t.Fatalf("error = %v, want metadata_malformed", err)
	}
}
