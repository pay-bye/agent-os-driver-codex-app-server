package transmit

import (
	"context"
	"time"
)

func waitForStart(
	ctx context.Context,
	started <-chan struct{},
	result <-chan turnOutcome,
) (turnOutcome, bool, error) {
	select {
	case item := <-result:
		return item, true, item.err
	case <-started:
		return turnOutcome{}, false, nil
	case <-ctx.Done():
		return turnOutcome{}, true, ctx.Err()
	}
}

func wait(ctx context.Context, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
