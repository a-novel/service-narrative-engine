package core_test

import (
	"context"
	"errors"
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
			// A gate always holds one slot.
			name: "Success/AtLeastOneSlot",

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
