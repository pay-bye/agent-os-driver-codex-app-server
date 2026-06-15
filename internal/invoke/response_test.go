package invoke

import (
	"context"
	"testing"
)

func TestClientTreatsEmptyClaimAsNoWork(t *testing.T) {
	server := newHTTPServer(t)
	server.empty = true
	defer server.Close()
	client := New(server.URL, server.Client())

	claim, err := client.Claim(context.Background(), "q01", "1a7c3e9b5f024d68", 60)

	if err != nil {
		t.Fatal(err)
	}
	if !claim.Empty {
		t.Fatalf("expected empty claim, got %+v", claim)
	}
}
