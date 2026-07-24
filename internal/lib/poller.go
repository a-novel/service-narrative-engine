package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/a-novel-kit/golib/logging"
)

// Poll runs fn on a fixed interval until ctx is cancelled.
//
// When fn reports it did work, Poll runs it again immediately rather than waiting out the
// interval, so a backlog drains at full speed instead of one item per tick. An error is logged
// and the loop continues: a transient failure reaching a dependency must not take a background
// worker down.
//
// stagger delays the first run only, and is deterministic rather than randomised. Several pollers
// sharing one interval de-synchronise without drawing from math/rand, which the security linter
// objects to and which would make an interleaving impossible to reproduce.
//
// name identifies the poller in its log lines. Every message is formatted into the message string
// with no fields, because the two logging presets read fields differently — the local one as
// printf operands, the Google Cloud one as structured attributes — and agree only when there are
// none.
func Poll(
	ctx context.Context,
	logger logging.Log,
	name string,
	interval, stagger time.Duration,
	fn func(context.Context) (bool, error),
) {
	if !sleep(ctx, stagger) {
		return
	}

	// The condition, rather than a bare for, is what stops the drain path below from running fn
	// again on a cancelled context.
	for ctx.Err() == nil {
		worked, err := fn(ctx)

		switch {
		case err != nil:
			logger.Err(ctx, fmt.Sprintf("poll %s: %v", name, err))
		case worked:
			// Worth a line of its own: it is the difference between a backlog draining and a
			// backlog nobody noticed, in a stack with no metrics path.
			logger.Info(ctx, fmt.Sprintf("poll %s: ran a unit of work", name))

			continue
		}

		if !sleep(ctx, interval) {
			return
		}
	}
}

// sleep waits out d, reporting whether the wait finished rather than the context being cancelled.
// A duration of zero or less returns immediately, still honouring an already-cancelled context.
func sleep(ctx context.Context, duration time.Duration) bool {
	if duration <= 0 {
		return ctx.Err() == nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
