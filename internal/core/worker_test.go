package core_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"
	jobsmocks "github.com/a-novel/service-jobs/pkg/go/mocks"
	jobworker "github.com/a-novel/service-jobs/pkg/go/worker"

	loggingpresets "github.com/a-novel-kit/golib/logging/presets"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
)

const workerTestTimeout = time.Second

func TestWorker(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")
	validConfig := config.Worker{
		Concurrency: 1, PollInterval: time.Millisecond, JobDeadline: time.Second,
		Lease: 2 * time.Second, DrainBudget: 2 * time.Second,
	}

	newTestWorker := func(
		t *testing.T,
		client *jobsmocks.MockClient,
		handlers map[string]jobworker.Handler,
		kinds []string,
		workerConfig config.Worker,
		logger *loggingpresets.LogLocal,
	) *core.Worker {
		t.Helper()

		executor, err := jobworker.NewExecutor(client, handlers)
		require.NoError(t, err)

		jobWorker, err := core.NewWorker(client, executor, kinds, workerConfig, logger)
		require.NoError(t, err)

		return jobWorker
	}

	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "Success/EmptyKindsIdleSilently",
			run: func(t *testing.T) {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				client := jobsmocks.NewMockClient(t)

				var logs bytes.Buffer

				client.EXPECT().
					JobClaim(mock.Anything, mock.MatchedBy(func(request *servicejobs.JobClaimRequest) bool {
						return request != nil && request.Kinds != nil && len(request.GetKinds()) == 0 &&
							request.GetWorkerId() != "" && request.GetLimit() == 1 && request.GetLeaseSeconds() == 2
					})).
					Run(func(context.Context, *servicejobs.JobClaimRequest, ...grpc.CallOption) {
						cancel()
					}).
					Return(&servicejobs.JobClaimResponse{}, nil).
					Once()

				jobWorker := newTestWorker(
					t, client, map[string]jobworker.Handler{}, []string{}, validConfig,
					&loggingpresets.LogLocal{Out: &logs},
				)
				jobWorker.Run(ctx)

				require.Empty(t, logs.String())
				client.AssertExpectations(t)
			},
		},
		{
			name: "Success/ExecutesClaimAndDrainsOnCancel",
			run: func(t *testing.T) {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				client := jobsmocks.NewMockClient(t)
				job := &servicejobs.Job{Id: "job-1", Kind: "generate"}
				started := make(chan struct{})
				release := make(chan struct{})
				workerDone := make(chan struct{})

				var handlerCalls atomic.Int64

				client.EXPECT().
					JobClaim(mock.Anything, mock.MatchedBy(func(request *servicejobs.JobClaimRequest) bool {
						return request != nil && len(request.GetKinds()) == 1 && request.GetKinds()[0] == "generate" &&
							request.GetWorkerId() != "" && request.GetLimit() == 1 && request.GetLeaseSeconds() == 2
					})).
					Return(&servicejobs.JobClaimResponse{Jobs: []*servicejobs.Job{job}}, nil).
					Once()
				client.EXPECT().
					JobSettle(mock.Anything, mock.MatchedBy(func(request *servicejobs.JobSettleRequest) bool {
						return request != nil && request.GetId() == job.GetId() && request.GetWorkerId() != ""
					})).
					Return(&servicejobs.JobSettleResponse{Job: &servicejobs.Job{
						Id: job.GetId(), Kind: job.GetKind(), Status: servicejobs.JobStatusSucceeded,
					}}, nil).
					Once()

				handlers := map[string]jobworker.Handler{
					"generate": func(
						ctx context.Context,
						_ *servicejobs.Job,
						_ jobworker.ProviderCallRecorder,
					) (json.RawMessage, error) {
						handlerCalls.Add(1)
						close(started)

						select {
						case <-release:
							return json.RawMessage(`{"ok":true}`), nil
						case <-ctx.Done():
							return nil, ctx.Err()
						}
					},
				}

				jobWorker := newTestWorker(
					t, client, handlers, []string{"generate"}, validConfig,
					&loggingpresets.LogLocal{Out: &bytes.Buffer{}},
				)

				go func() {
					defer close(workerDone)

					jobWorker.Run(ctx)
				}()

				select {
				case <-started:
				case <-time.After(workerTestTimeout):
					require.FailNow(t, "worker did not execute the claimed job")
				}

				cancel()

				select {
				case <-workerDone:
					require.FailNow(t, "worker returned before the in-flight job drained")
				case <-time.After(10 * time.Millisecond):
				}

				close(release)

				select {
				case <-workerDone:
				case <-time.After(workerTestTimeout):
					require.FailNow(t, "worker did not return after the in-flight job drained")
				}

				require.Equal(t, int64(1), handlerCalls.Load())
				client.AssertExpectations(t)
			},
		},
		{
			name: "Success/ContinuesAfterClaimFailure",
			run: func(t *testing.T) {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				client := jobsmocks.NewMockClient(t)

				var (
					claims atomic.Int64
					logs   bytes.Buffer
				)

				client.EXPECT().
					JobClaim(mock.Anything, mock.Anything).
					RunAndReturn(func(
						context.Context,
						*servicejobs.JobClaimRequest,
						...grpc.CallOption,
					) (*servicejobs.JobClaimResponse, error) {
						if claims.Add(1) == 1 {
							return nil, errFoo
						}

						cancel()

						return &servicejobs.JobClaimResponse{}, nil
					}).
					Twice()

				jobWorker := newTestWorker(
					t, client, map[string]jobworker.Handler{}, []string{}, validConfig,
					&loggingpresets.LogLocal{Out: &logs},
				)
				jobWorker.Run(ctx)

				require.Equal(t, int64(2), claims.Load())
				require.Contains(t, logs.String(), "claim narrative job: foo")
				client.AssertExpectations(t)
			},
		},
		{
			name: "Error/InvalidConfig",
			run: func(t *testing.T) {
				t.Helper()

				client := jobsmocks.NewMockClient(t)
				executor, err := jobworker.NewExecutor(client, nil)
				require.NoError(t, err)

				jobWorker, err := core.NewWorker(
					client,
					executor,
					nil,
					config.Worker{},
					&loggingpresets.LogLocal{Out: &bytes.Buffer{}},
				)
				require.ErrorIs(t, err, core.ErrInvalidWorkerConfig)
				require.Nil(t, jobWorker)
				client.AssertExpectations(t)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testCase.run(t)
		})
	}
}
