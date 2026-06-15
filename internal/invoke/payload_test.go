package invoke

import (
	"context"
	"testing"
)

func TestClientReturnsClaimPayload(t *testing.T) {
	server := newHTTPServer(t)
	defer server.Close()
	client := New(server.URL, server.Client())

	claim, err := client.Claim(context.Background(), "q01", "1a7c3e9b5f024d68", 60)

	if err != nil {
		t.Fatal(err)
	}
	work, ok := claim.Payload["work"].(map[string]any)
	if !ok || work["prompt"] != "run close" {
		t.Fatalf("unexpected payload %+v", claim.Payload)
	}
}
