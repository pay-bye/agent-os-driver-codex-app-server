package transmit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/invoke"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/status"
)

func TestLeaseIDUsesOpaqueShape(t *testing.T) {
	value := Runner{}.leaseID()

	requireHexLength(t, value, 32)
}

func TestStepTurnsClaimedWorkIntoAck(t *testing.T) {
	invocation := newInvocationServer(t, claimWithPayload(map[string]any{"work": map[string]any{"prompt": "run close"}}))
	defer invocation.Close()
	app := newAppServer(t)
	defer app.Close()

	counts := &status.Counts{}
	runner := Runner{
		Config:        validConfig(invocation.URL, app.Endpoint()),
		Invocation:    invoke.New(invocation.URL, invocation.Client()),
		Control:       app,
		Counts:        counts,
		Compatibility: successfulCheck(),
		NewLeaseID:    func() string { return "0c4a72f19d8e5b30" },
		Now:           func() time.Time { return time.Unix(1, 0).UTC() },
	}

	outcome, err := runner.Step(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != Acked {
		t.Fatalf("expected acked outcome, got %+v", outcome)
	}
	requireRoutes(t, invocation.Routes(), []string{"/claim", "/extend", "/ack"})
	requireMethods(t, app.Methods(), []string{"thread/start", "turn/start"})
	if counts.Acks != 1 || counts.Extensions != 1 {
		t.Fatalf("unexpected counts %+v", counts)
	}
	if invocation.AckToken() != "x-token" {
		t.Fatalf("ack token = %q, want x-token", invocation.AckToken())
	}
	if invocation.ExtensionToken() != "x-token" {
		t.Fatalf("extension token = %q, want x-token", invocation.ExtensionToken())
	}
}

func TestStepNacksInvalidPayloadWithFailureNeeds(t *testing.T) {
	invocation := newInvocationServer(t, claimWithPayload(map[string]any{"work": map[string]any{"prompt": 42}}))
	defer invocation.Close()
	app := newAppServer(t)
	defer app.Close()
	runner := Runner{
		Config:        validConfig(invocation.URL, app.Endpoint()),
		Invocation:    invoke.New(invocation.URL, invocation.Client()),
		Control:       app,
		Counts:        &status.Counts{},
		Compatibility: successfulCheck(),
		NewLeaseID:    func() string { return "0c4a72f19d8e5b30" },
		Now:           func() time.Time { return time.Unix(1, 0).UTC() },
	}

	outcome, err := runner.Step(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != Nacked {
		t.Fatalf("expected nacked outcome, got %+v", outcome)
	}
	requireRoutes(t, invocation.Routes(), []string{"/claim", "/nack"})
	if invocation.NackNeed() != "failed" {
		t.Fatalf("expected failure need, got %q", invocation.NackNeed())
	}
	if invocation.NackToken() != "x-token" {
		t.Fatalf("nack token = %q, want x-token", invocation.NackToken())
	}
	requireMethods(t, app.Methods(), nil)
}

func TestStepNacksAppError(t *testing.T) {
	invocation := newInvocationServer(t, claimWithPayload(map[string]any{"work": map[string]any{"prompt": "run close"}}))
	defer invocation.Close()
	app := newAppServer(t)
	app.failTurn = true
	defer app.Close()
	runner := Runner{
		Config:        validConfig(invocation.URL, app.Endpoint()),
		Invocation:    invoke.New(invocation.URL, invocation.Client()),
		Control:       app,
		Counts:        &status.Counts{},
		Compatibility: successfulCheck(),
		NewLeaseID:    func() string { return "0c4a72f19d8e5b30" },
		Now:           func() time.Time { return time.Unix(1, 0).UTC() },
	}

	outcome, err := runner.Step(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != Nacked {
		t.Fatalf("expected nacked outcome, got %+v", outcome)
	}
	requireRoutes(t, invocation.Routes(), []string{"/claim", "/extend", "/nack"})
}

func TestStepSendsGeneratedOpaqueLeaseID(t *testing.T) {
	invocation := newInvocationServer(t, claimWithPayload(map[string]any{"work": map[string]any{"prompt": "run close"}}))
	defer invocation.Close()
	app := newAppServer(t)
	defer app.Close()
	runner := Runner{
		Config:        validConfig(invocation.URL, app.Endpoint()),
		Invocation:    invoke.New(invocation.URL, invocation.Client()),
		Control:       app,
		Counts:        &status.Counts{},
		Compatibility: successfulCheck(),
	}

	_, err := runner.Step(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	requireHexLength(t, invocation.ClaimID(), 32)
}

func TestStepBacksOffOnEmptyClaim(t *testing.T) {
	invocation := newInvocationServer(t, map[string]any{"empty": true})
	defer invocation.Close()
	app := newAppServer(t)
	defer app.Close()
	runner := Runner{
		Config:        validConfig(invocation.URL, app.Endpoint()),
		Invocation:    invoke.New(invocation.URL, invocation.Client()),
		Control:       app,
		Counts:        &status.Counts{},
		Compatibility: successfulCheck(),
		NewLeaseID:    func() string { return "0c4a72f19d8e5b30" },
		Now:           func() time.Time { return time.Unix(1, 0).UTC() },
	}

	outcome, err := runner.Step(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != Empty {
		t.Fatalf("expected empty outcome, got %+v", outcome)
	}
	requireRoutes(t, invocation.Routes(), []string{"/claim"})
	requireMethods(t, app.Methods(), nil)
}

func TestStepStopsBeforeClaimWhenCompatibilityFails(t *testing.T) {
	invocation := newInvocationServer(t, claimWithPayload(map[string]any{"work": map[string]any{"prompt": "run close"}}))
	defer invocation.Close()
	app := newAppServer(t)
	defer app.Close()
	counts := &status.Counts{}
	runner := Runner{
		Config:        validConfig(invocation.URL, app.Endpoint()),
		Invocation:    invoke.New(invocation.URL, invocation.Client()),
		Control:       app,
		Counts:        counts,
		Compatibility: failingCheck{err: compatibility.ErrRouteMissing},
		NewLeaseID:    func() string { return "0c4a72f19d8e5b30" },
	}

	_, err := runner.Step(context.Background())

	if !errors.Is(err, compatibility.ErrRouteMissing) {
		t.Fatalf("error = %v, want route_missing", err)
	}
	if counts.LastErrorCode != "route_missing" {
		t.Fatalf("error code = %q, want route_missing", counts.LastErrorCode)
	}
	requireRoutes(t, invocation.Routes(), nil)
	requireMethods(t, app.Methods(), nil)
}
