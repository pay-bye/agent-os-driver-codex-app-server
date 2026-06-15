package invoke

import (
	"context"
	"testing"
	"time"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
)

func TestClientClaimsExtendsAndAcks(t *testing.T) {
	server := newHTTPServer(t)
	defer server.Close()
	client := New(server.URL, server.Client())

	claim, err := client.Claim(context.Background(), "q01", "1a7c3e9b5f024d68", 60)
	if err != nil {
		t.Fatal(err)
	}
	if claim.Empty || claim.WorkItemID != "2b8d4f0a6c93e157" {
		t.Fatalf("unexpected claim %+v", claim)
	}
	if claim.Token != "x-token" {
		t.Fatalf("claim token = %q, want x-token", claim.Token)
	}
	if err := client.Extend(context.Background(), "1a7c3e9b5f024d68", claim.Token, time.Unix(100, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	if err := client.Ack(context.Background(), "1a7c3e9b5f024d68", claim.Token, []config.Need{{Kind: "done"}}); err != nil {
		t.Fatal(err)
	}

	requireRoutes(t, server.Routes(), []string{"/claim", "/extend", "/ack"})
	if server.Token("/extend") != "x-token" {
		t.Fatalf("extend token = %q, want x-token", server.Token("/extend"))
	}
	if server.Token("/ack") != "x-token" {
		t.Fatalf("ack token = %q, want x-token", server.Token("/ack"))
	}
}

func TestClientSendsFailureNeedsOnNack(t *testing.T) {
	server := newHTTPServer(t)
	defer server.Close()
	client := New(server.URL, server.Client())

	err := client.Nack(
		context.Background(),
		"1a7c3e9b5f024d68",
		"x-token",
		Payload{"error_code": "invalid_payload"},
		[]config.Need{{Kind: "failed"}},
	)

	if err != nil {
		t.Fatal(err)
	}
	if server.NackNeed() != "failed" {
		t.Fatalf("expected failure need, got %q", server.NackNeed())
	}
	if server.Token("/nack") != "x-token" {
		t.Fatalf("nack token = %q, want x-token", server.Token("/nack"))
	}
}
