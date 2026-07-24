package lib_test

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	loggingpresets "github.com/a-novel-kit/golib/logging/presets"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

const (
	// pollTimeout bounds every wait below. Each case drives Poll with an interval of an hour, so
	// anything that has not happened inside a second is not going to happen because it was slow.
	pollTimeout = time.Second

	// pollStagger is long enough to measure without making the suite wait on it.
	pollStagger = 150 * time.Millisecond

	// pollDrainRuns is how many consecutive units of work the drain case reports.
	pollDrainRuns = 5
)

// discardLogger keeps the poller's own log lines out of the test output. No case asserts on them:
// what is under test is the loop, and the lines are covered by the shape of the calls it makes.
var discardLogger = &loggingpresets.LogLocal{Out: io.Discard}

func TestPoll(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	t.Run("Success/ReturnsOnCancel", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		returned := make(chan struct{})

		go func() {
			defer close(returned)

			lib.Poll(ctx, discardLogger, "test", time.Hour, 0, func(context.Context) (bool, error) {
				return false, nil
			})
		}()

		cancel()

		select {
		case <-returned:
		case <-time.After(pollTimeout):
			require.FailNow(t, "Poll did not return on context cancellation")
		}
	})

	t.Run("Success/RerunsImmediatelyAfterWork", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		var calls atomic.Int64

		drained := make(chan struct{})

		//nolint:unparam // The callback's signature is dictated by lib.Poll, not by this test.
		drain := func(context.Context) (bool, error) {
			count := calls.Add(1)
			if count < pollDrainRuns {
				return true, nil
			}

			if count == pollDrainRuns {
				close(drained)
			}

			return false, nil
		}

		// The interval is an hour, so reaching the last run inside pollTimeout is only possible if
		// reporting work skips the wait entirely. That is the whole behaviour under test.
		go lib.Poll(ctx, discardLogger, "test", time.Hour, 0, drain)

		select {
		case <-drained:
		case <-time.After(pollTimeout):
			require.FailNow(t, "Poll waited out its interval instead of re-running after work")
		}
	})

	t.Run("Success/ContinuesAfterError", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		var calls atomic.Int64

		recovered := make(chan struct{})

		//nolint:unparam // The callback's signature is dictated by lib.Poll, not by this test.
		failOnce := func(context.Context) (bool, error) {
			switch calls.Add(1) {
			case 1:
				return false, errFoo
			case 2:
				close(recovered)
			}

			return false, nil
		}

		go lib.Poll(ctx, discardLogger, "test", time.Millisecond, 0, failOnce)

		select {
		case <-recovered:
		case <-time.After(pollTimeout):
			require.FailNow(t, "Poll stopped after the callback returned an error")
		}
	})

	t.Run("Success/StaggersTheFirstRun", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		started := time.Now()
		firstCall := make(chan time.Duration, 1)

		//nolint:unparam // The callback's signature is dictated by lib.Poll, not by this test.
		recordFirstCall := func(context.Context) (bool, error) {
			select {
			case firstCall <- time.Since(started):
			default:
			}

			return false, nil
		}

		go lib.Poll(ctx, discardLogger, "test", time.Hour, pollStagger, recordFirstCall)

		select {
		case elapsed := <-firstCall:
			require.GreaterOrEqual(t, elapsed, pollStagger)
		case <-time.After(pollTimeout):
			require.FailNow(t, "Poll never ran its callback")
		}
	})
}
