package transmit

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var errExtensionFailed = errors.New("extension_failed")

type leaseRenewal struct {
	runner    Runner
	leaseID   string
	token     string
	expiresAt time.Time
}

func (r *leaseRenewal) Extend(ctx context.Context) error {
	nextExpiresAt := r.expiresAt.Add(r.runner.leaseDuration())
	if err := r.runner.Invocation.Extend(ctx, r.leaseID, r.token, nextExpiresAt); err != nil {
		r.runner.counts().LastErrorCode = "extend_failed"
		return err
	}
	r.expiresAt = nextExpiresAt
	r.runner.counts().Extensions++
	return nil
}

func wrapExtension(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", errExtensionFailed, err)
}
