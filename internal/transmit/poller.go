package transmit

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/control"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/invoke"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/status"
	"time"
)

const (
	Empty           OutcomeKind = "empty"
	Acked           OutcomeKind = "acked"
	Nacked          OutcomeKind = "nacked"
	ExtensionFailed OutcomeKind = "extension_failed"
)

const generatedIDBytes = 16

type OutcomeKind string

type Outcome struct {
	Kind OutcomeKind
}

type Control interface {
	StartThread(context.Context, string) (string, error)
	RunTurn(context.Context, control.TurnRequest) (control.Turn, error)
	InterruptTurn(context.Context, string) error
}

type Check interface {
	Verify(context.Context) (compatibility.Result, error)
}

type turnOutcome struct {
	turn control.Turn
	err  error
}

type Runner struct {
	Config          config.Config
	Invocation      *invoke.Client
	Control         Control
	Counts          *status.Counts
	Compatibility   Check
	NewLeaseID      func() string
	Now             func() time.Time
	RenewalInterval time.Duration
}

func (r Runner) Step(ctx context.Context) (Outcome, error) {
	if err := r.checkCompatibility(ctx); err != nil {
		return Outcome{}, err
	}
	counts := r.counts()
	counts.ClaimAttempts++

	claim, err := r.Invocation.Claim(ctx, r.Config.ChannelKey, r.leaseID(), r.Config.LeaseSeconds)
	if err != nil {
		return Outcome{}, err
	}
	if claim.Empty {
		counts.EmptyClaims++
		return Outcome{Kind: Empty}, nil
	}
	return r.transmitClaim(ctx, claim)
}

func (r Runner) checkCompatibility(ctx context.Context) error {
	if r.Compatibility == nil {
		return nil
	}
	result, err := r.Compatibility.Verify(ctx)
	if err != nil {
		r.counts().LastErrorCode = result.DiagnosticCode
		return err
	}
	return nil
}

func (r Runner) counts() *status.Counts {
	if r.Counts != nil {
		return r.Counts
	}
	return &status.Counts{}
}

func (r Runner) leaseID() string {
	if r.NewLeaseID != nil {
		return r.NewLeaseID()
	}
	return generatedID()
}

func (r Runner) transmitClaim(ctx context.Context, claim invoke.Claim) (Outcome, error) {
	counts := r.counts()
	counts.ActiveLeaseID = claim.LeaseID
	counts.WorkItemID = claim.WorkItemID

	input, err := config.ExtractPromptText(claim.Payload, r.Config.InputTextPointer)
	if err != nil {
		return r.nack(ctx, claim, "invalid_payload")
	}
	threadID, err := r.Control.StartThread(ctx, r.Config.WorkspaceRoot)
	if err != nil {
		return r.nack(ctx, claim, "app_unreachable")
	}
	counts.ThreadID = threadID

	turn, err := r.runTurn(ctx, claim, threadID, input)
	if errors.Is(err, errExtensionFailed) {
		return Outcome{Kind: ExtensionFailed}, err
	}
	if err != nil {
		return r.nack(ctx, claim, "app_error")
	}
	counts.TurnID = turn.ID
	if !turn.Completed() {
		return r.nack(ctx, claim, "turn_failed")
	}
	return r.ack(ctx, claim)
}

func (r Runner) nack(ctx context.Context, claim invoke.Claim, code string) (Outcome, error) {
	counts := r.counts()
	if err := r.Invocation.Nack(ctx, claim.LeaseID, claim.Token, status.FailurePayload(code), r.Config.FailureNeeds); err != nil {
		return Outcome{}, err
	}
	counts.Nacks++
	counts.LastErrorCode = code
	return Outcome{Kind: Nacked}, nil
}

func (r Runner) runTurn(
	ctx context.Context,
	claim invoke.Claim,
	threadID string,
	input string,
) (control.Turn, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	renewal := r.leaseRenewal(claim)
	started := make(chan struct{})
	result := make(chan turnOutcome, 1)
	go r.startTurn(ctx, threadID, input, renewal, started, result)

	return r.waitForTurn(ctx, renewal, started, result, cancel)
}

func (r Runner) leaseRenewal(claim invoke.Claim) *leaseRenewal {
	expiresAt := claim.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = r.now()
	}
	return &leaseRenewal{
		runner:    r,
		leaseID:   claim.LeaseID,
		token:     claim.Token,
		expiresAt: expiresAt,
	}
}

func (r Runner) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now().UTC()
}

func (r Runner) startTurn(
	ctx context.Context,
	threadID string,
	input string,
	renewal *leaseRenewal,
	started chan<- struct{},
	result chan<- turnOutcome,
) {
	turn, err := r.Control.RunTurn(ctx, control.TurnRequest{
		ThreadID: threadID,
		Input:    input,
		OnStarted: func(turn control.Turn) error {
			r.counts().TurnID = turn.ID
			err := renewal.Extend(ctx)
			if err != nil {
				r.interruptActiveTurn()
			}
			close(started)
			return wrapExtension(err)
		},
	})
	result <- turnOutcome{turn: turn, err: err}
}

func (r Runner) interruptActiveTurn() {
	turnID := r.counts().TurnID
	if turnID == "" || r.Control == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Control.InterruptTurn(ctx, turnID); err != nil {
		r.counts().LastErrorCode = "turn_interrupt_failed"
	}
}

func (r Runner) waitForTurn(
	ctx context.Context,
	renewal *leaseRenewal,
	started <-chan struct{},
	result <-chan turnOutcome,
	cancel context.CancelFunc,
) (control.Turn, error) {
	if item, done, err := waitForStart(ctx, started, result); done {
		return item.turn, err
	}
	return r.renewUntilDone(ctx, renewal, result, cancel)
}

func (r Runner) renewUntilDone(
	ctx context.Context,
	renewal *leaseRenewal,
	result <-chan turnOutcome,
	cancel context.CancelFunc,
) (control.Turn, error) {
	ticker := time.NewTicker(r.renewalInterval())
	defer ticker.Stop()
	for {
		select {
		case item := <-result:
			return item.turn, item.err
		case <-ticker.C:
			if err := renewal.Extend(ctx); err != nil {
				r.interruptActiveTurn()
				cancel()
				return control.Turn{}, wrapExtension(err)
			}
		case <-ctx.Done():
			r.interruptActiveTurn()
			cancel()
			return control.Turn{}, ctx.Err()
		}
	}
}

func (r Runner) renewalInterval() time.Duration {
	if r.RenewalInterval > 0 {
		return r.RenewalInterval
	}
	duration := r.leaseDuration() / 2
	if duration <= 0 {
		return time.Second
	}
	return duration
}

func (r Runner) leaseDuration() time.Duration {
	return time.Duration(r.Config.LeaseSeconds) * time.Second
}

func (r Runner) ack(ctx context.Context, claim invoke.Claim) (Outcome, error) {
	counts := r.counts()
	if err := r.Invocation.Ack(ctx, claim.LeaseID, claim.Token, r.Config.CompletionNeeds); err != nil {
		return Outcome{}, err
	}
	counts.Acks++
	return Outcome{Kind: Acked}, nil
}

func (r Runner) Run(ctx context.Context, idleDelay time.Duration) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		outcome, err := r.Step(ctx)
		if err != nil {
			return err
		}
		if outcome.Kind == Empty {
			wait(ctx, idleDelay)
		}
	}
}

func (r Runner) Validate() error {
	if r.Invocation == nil {
		return errors.New("invocation_client_missing")
	}
	if r.Control == nil {
		return errors.New("control_client_missing")
	}
	return nil
}

func generatedID() string {
	value := make([]byte, generatedIDBytes)
	if _, err := rand.Read(value); err != nil {
		return fallbackID(time.Now().UTC())
	}
	return hex.EncodeToString(value)
}

func fallbackID(value time.Time) string {
	sum := sha256.Sum256([]byte(value.Format(time.RFC3339Nano)))
	return hex.EncodeToString(sum[:generatedIDBytes])
}
