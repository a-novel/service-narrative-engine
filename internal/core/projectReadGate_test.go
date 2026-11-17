package core_test

import (
	"context"
	"errors"
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

func TestProjectReadGate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		// size is the size the gate is built with.
		size int
		// held is how many reads hold a slot of the gate while the read under test starts.
		held int
		// cancel ends the context of the read under test before it starts.
		cancel  bool
		readErr error

		expectRan bool
		expectErr error
	}{
		{
			name: "Success",

			size: 2,

			expectRan: true,
		},
		{
			// Without GOMEMLIMIT, as go test runs, a gate of size 0 runs as many reads as CPUs.
			name: "Success/SizedFromCPUs",

			expectRan: true,
		},
		{
			name: "Success/FreeSlot",

			size: 2,
			held: 1,

			expectRan: true,
		},
		{
			name: "Error/Read",

			size:    2,
			readErr: errFoo,

			expectRan: true,
			expectErr: errFoo,
		},
		{
			// A read past the bound waits for a slot, and gives up when its context ends.
			name: "Error/Full",

			size:   2,
			held:   2,
			cancel: true,

			expectErr: context.Canceled,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gate := core.NewProjectReadGate(testCase.size)
			release := make(chan struct{})
			holding := make(chan struct{}, testCase.held)
			done := make(chan error, testCase.held)

			for range testCase.held {
				go func() {
					done <- gate.Run(t.Context(), func(context.Context) error {
						holding <- struct{}{}

						<-release

						return nil
					})
				}()
			}

			for range testCase.held {
				<-holding
			}

			ctx, cancel := context.WithCancel(t.Context())
			if testCase.cancel {
				cancel()
			}

			var ran bool

			err := gate.Run(ctx, func(context.Context) error {
				ran = true

				return testCase.readErr
			})

			cancel()
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expectRan, ran)

			// The holders free their slots once they return, so a later read runs.
			close(release)

			for range testCase.held {
				require.NoError(t, <-done)
			}

			waitCtx, waitCancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer waitCancel()

			require.NoError(t, gate.Run(waitCtx, func(context.Context) error { return nil }))
		})
	}
}

// TestProjectReadGateMemory sizes a gate from a 1 GiB memory limit: the reads take half of it, at 128 MiB each, so
// the gate runs 4 at once and a fifth waits.
//
//nolint:paralleltest // debug.SetMemoryLimit sets the limit of the whole process.
func TestProjectReadGateMemory(t *testing.T) {
	previous := debug.SetMemoryLimit(1 << 30)
	gate := core.NewProjectReadGate(0)

	debug.SetMemoryLimit(previous)

	for range 4 {
		t.Cleanup(holdGate(t, gate))
	}

	// A fifth slot would run the read at once; with four, the read waits past its deadline.
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	var ran bool

	err := gate.Run(ctx, func(context.Context) error {
		ran = true

		return nil
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.False(t, ran)
}
