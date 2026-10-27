package checker

import (
	"context"
	"fmt"
	"time"
)

// Runtime reports process uptime as a liveness signal.
type Runtime struct {
	startedAt time.Time
}

// NewRuntime creates a checker that has been alive since startedAt.
func NewRuntime(startedAt time.Time) *Runtime {
	return &Runtime{startedAt: startedAt}
}

// Name implements domain.Checker.
func (r *Runtime) Name() string {
	return "runtime"
}

// Check implements domain.Checker.
func (r *Runtime) Check(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return fmt.Sprintf("uptime=%s", time.Since(r.startedAt).Round(time.Second)), nil
	}
}
