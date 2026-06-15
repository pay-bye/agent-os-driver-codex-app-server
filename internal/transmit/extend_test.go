package transmit

import (
	"context"
	"testing"
	"time"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/invoke"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/status"
)

func TestStepExtendsLeaseBeforeDelayedCompletionAck(t *testing.T) {
	invocation := newInvocationServer(t, claimWithPayload(map[string]any{"work": map[string]any{"prompt": "run close"}}))
	defer invocation.Close()
	app := newAppServer(t)
	app.delayedCompletion = true
	app.completionGate = invocation.Extended()
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
	if outcome.Kind != Acked {
		t.Fatalf("expected acked outcome, got %+v", outcome)
	}
	requireRoutes(t, invocation.Routes(), []string{"/claim", "/extend", "/ack"})
}

func TestStepRenewsLeaseUntilDelayedCompletionAck(t *testing.T) {
	invocation := newInvocationServer(t, claimWithPayload(map[string]any{"work": map[string]any{"prompt": "run close"}}))
	invocation.extendGoal = 2
	defer invocation.Close()
	app := newAppServer(t)
	app.delayedCompletion = true
	app.completionGate = invocation.Extended()
	defer app.Close()
	counts := &status.Counts{}
	runner := Runner{
		Config:          validConfig(invocation.URL, app.Endpoint()),
		Invocation:      invoke.New(invocation.URL, invocation.Client()),
		Control:         app,
		Counts:          counts,
		Compatibility:   successfulCheck(),
		NewLeaseID:      func() string { return "0c4a72f19d8e5b30" },
		Now:             func() time.Time { return time.Unix(1, 0).UTC() },
		RenewalInterval: 10 * time.Millisecond,
	}

	outcome, err := runner.Step(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != Acked {
		t.Fatalf("expected acked outcome, got %+v", outcome)
	}
	if counts.Extensions != 2 {
		t.Fatalf("expected two lease extensions, got %+v", counts)
	}
	requireRoutes(t, invocation.Routes(), []string{"/claim", "/extend", "/extend", "/ack"})
}

func TestStepStopsWhenExtendFails(t *testing.T) {
	invocation := newInvocationServer(t, claimWithPayload(map[string]any{"work": map[string]any{"prompt": "run close"}}))
	invocation.failExtend = true
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

	if err == nil {
		t.Fatal("expected extend failure")
	}
	if outcome.Kind != ExtensionFailed {
		t.Fatalf("expected extension failure outcome, got %+v", outcome)
	}
	requireRoutes(t, invocation.Routes(), []string{"/claim", "/extend"})
	requireMethods(t, app.Methods(), []string{"thread/start", "turn/start", "turn/interrupt"})
}
