package transmit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/invoke"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/status"
)

func TestStepInterruptsActiveTurnWhenContextCancels(t *testing.T) {
	invocation := newInvocationServer(t, claimWithPayload(map[string]any{"work": map[string]any{"prompt": "run close"}}))
	defer invocation.Close()
	app := newAppServer(t)
	app.delayedCompletion = true
	app.completionGate = make(chan struct{})
	defer app.Close()
	ctx, cancel := context.WithCancel(context.Background())
	app.cancelAfterStart = cancel
	runner := Runner{
		Config:        validConfig(invocation.URL, app.Endpoint()),
		Invocation:    invoke.New(invocation.URL, invocation.Client()),
		Control:       app,
		Counts:        &status.Counts{},
		Compatibility: successfulCheck(),
		NewLeaseID:    func() string { return "0c4a72f19d8e5b30" },
		Now:           func() time.Time { return time.Unix(1, 0).UTC() },
	}

	_, err := runner.Step(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	requireRoutes(t, invocation.Routes(), []string{"/claim", "/extend"})
	requireMethods(t, app.Methods(), []string{"thread/start", "turn/start", "turn/interrupt"})
}
