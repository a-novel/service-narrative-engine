package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"
	jobsmocks "github.com/a-novel/service-jobs/pkg/go/mocks"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
)

func TestJobExecute(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")
	errLostClaim := status.Error(codes.FailedPrecondition, "job not claimed by this worker")
	result := json.RawMessage(`{"manuscript_id":"00000000-0000-0000-0000-000000000007"}`)
	claimedJob := &servicejobs.Job{
		Id:          "00000000-0000-0000-0000-000000000001",
		Kind:        "narrative.generate",
		Payload:     json.RawMessage(`{"owner_id":"00000000-0000-0000-0000-000000000099"}`),
		OwnerId:     testActor.UserID.String(),
		Status:      servicejobs.JobStatusClaimed,
		Attempt:     1,
		MaxAttempts: 2,
	}
	runningProviderJob := &servicejobs.Job{
		Id:             "00000000-0000-0000-0000-000000000002",
		Kind:           "narrative.generate",
		Payload:        json.RawMessage(`{"idea_id":"00000000-0000-0000-0000-000000000003"}`),
		OwnerId:        testActor.UserID.String(),
		Status:         servicejobs.JobStatusClaimed,
		Attempt:        2,
		MaxAttempts:    2,
		ProviderCallId: "provider-operation-42",
	}
	unknownJob := &servicejobs.Job{
		Id:          "00000000-0000-0000-0000-000000000004",
		Kind:        "unknown",
		OwnerId:     testActor.UserID.String(),
		Status:      servicejobs.JobStatusClaimed,
		Attempt:     1,
		MaxAttempts: 1,
	}
	workerID := "narrative-rest-1"

	type handlerMock struct {
		kind   string
		handle func(
			t *testing.T,
			ctx context.Context,
			job *servicejobs.Job,
			recorder core.ProviderCallRecorder,
		) (json.RawMessage, error)
	}

	type settleMock struct {
		request  *servicejobs.JobSettleRequest
		response *servicejobs.JobSettleResponse
		err      error
	}

	type recordProviderCallMock struct {
		request  *servicejobs.JobRecordProviderCallRequest
		response *servicejobs.JobRecordProviderCallResponse
		err      error
	}

	testCases := []struct {
		name string

		request *core.JobExecuteRequest

		handlerMock            *handlerMock
		settleMock             *settleMock
		recordProviderCallMock *recordProviderCallMock
		expectLostClaimLog     bool

		expect    *servicejobs.Job
		expectErr error
	}{
		{
			name: "Success/Ownership",
			request: &core.JobExecuteRequest{
				Job: claimedJob, WorkerID: workerID, Deadline: time.Second,
			},
			handlerMock: &handlerMock{
				kind: "narrative.generate",
				handle: func(
					t *testing.T, _ context.Context, job *servicejobs.Job, _ core.ProviderCallRecorder,
				) (json.RawMessage, error) {
					t.Helper()
					require.Equal(t, testActor.UserID.String(), job.GetOwnerId())
					require.NotContains(t, string(job.GetPayload()), job.GetOwnerId())

					return result, nil
				},
			},
			settleMock: &settleMock{
				request: &servicejobs.JobSettleRequest{
					Id:       claimedJob.GetId(),
					WorkerId: workerID,
					Outcome:  &servicejobs.JobSettleResult{Result: result},
				},
				response: &servicejobs.JobSettleResponse{Job: &servicejobs.Job{
					Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusSucceeded,
				}},
			},
			expect: &servicejobs.Job{
				Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusSucceeded,
			},
		},
		{
			name: "Success/ProviderCallRecorded",
			request: &core.JobExecuteRequest{
				Job: claimedJob, WorkerID: workerID, Deadline: time.Second,
			},
			handlerMock: &handlerMock{
				kind: "narrative.generate",
				handle: func(
					t *testing.T, _ context.Context, _ *servicejobs.Job, recorder core.ProviderCallRecorder,
				) (json.RawMessage, error) {
					t.Helper()
					require.NoError(t, recorder("provider-operation-7"))

					return result, nil
				},
			},
			recordProviderCallMock: &recordProviderCallMock{
				request: &servicejobs.JobRecordProviderCallRequest{
					Id:             claimedJob.GetId(),
					WorkerId:       workerID,
					ProviderCallId: "provider-operation-7",
				},
				response: &servicejobs.JobRecordProviderCallResponse{Job: claimedJob},
			},
			settleMock: &settleMock{
				request: &servicejobs.JobSettleRequest{
					Id:       claimedJob.GetId(),
					WorkerId: workerID,
					Outcome:  &servicejobs.JobSettleResult{Result: result},
				},
				response: &servicejobs.JobSettleResponse{Job: &servicejobs.Job{
					Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusSucceeded,
				}},
			},
			expect: &servicejobs.Job{
				Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusSucceeded,
			},
		},
		{
			name: "Success/ReattachProviderCall",
			request: &core.JobExecuteRequest{
				Job: runningProviderJob, WorkerID: workerID, Deadline: time.Second,
			},
			handlerMock: &handlerMock{
				kind: "narrative.generate",
				handle: func(
					t *testing.T, _ context.Context, job *servicejobs.Job, _ core.ProviderCallRecorder,
				) (json.RawMessage, error) {
					t.Helper()
					require.Equal(t, "provider-operation-42", job.GetProviderCallId())

					return result, nil
				},
			},
			settleMock: &settleMock{
				request: &servicejobs.JobSettleRequest{
					Id:       runningProviderJob.GetId(),
					WorkerId: workerID,
					Outcome:  &servicejobs.JobSettleResult{Result: result},
				},
				response: &servicejobs.JobSettleResponse{Job: &servicejobs.Job{
					Id:     runningProviderJob.GetId(),
					Kind:   runningProviderJob.GetKind(),
					Status: servicejobs.JobStatusSucceeded,
				}},
			},
			expect: &servicejobs.Job{
				Id:     runningProviderJob.GetId(),
				Kind:   runningProviderJob.GetKind(),
				Status: servicejobs.JobStatusSucceeded,
			},
		},
		{
			name: "Failure/ProviderCallRecord",
			request: &core.JobExecuteRequest{
				Job: claimedJob, WorkerID: workerID, Deadline: time.Second,
			},
			handlerMock: &handlerMock{
				kind: "narrative.generate",
				handle: func(
					_ *testing.T, _ context.Context, _ *servicejobs.Job, recorder core.ProviderCallRecorder,
				) (json.RawMessage, error) {
					return nil, recorder("provider-operation-8")
				},
			},
			recordProviderCallMock: &recordProviderCallMock{
				request: &servicejobs.JobRecordProviderCallRequest{
					Id:             claimedJob.GetId(),
					WorkerId:       workerID,
					ProviderCallId: "provider-operation-8",
				},
				err: errFoo,
			},
			settleMock: &settleMock{
				request: &servicejobs.JobSettleRequest{
					Id:       claimedJob.GetId(),
					WorkerId: workerID,
					Outcome: &servicejobs.JobSettleFailure{Failure: &servicejobs.JobFailure{
						Error: json.RawMessage(`{"message":"record provider call: foo"}`),
					}},
				},
				response: &servicejobs.JobSettleResponse{Job: &servicejobs.Job{
					Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusFailed,
				}},
			},
			expect: &servicejobs.Job{
				Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusFailed,
			},
		},
		{
			name: "Failure/Retryable",
			request: &core.JobExecuteRequest{
				Job: claimedJob, WorkerID: workerID, Deadline: time.Second,
			},
			handlerMock: &handlerMock{
				kind: "narrative.generate",
				handle: func(
					_ *testing.T, _ context.Context, _ *servicejobs.Job, _ core.ProviderCallRecorder,
				) (json.RawMessage, error) {
					return nil, errors.Join(errFoo, core.ErrJobRetryable)
				},
			},
			settleMock: &settleMock{
				request: &servicejobs.JobSettleRequest{
					Id:       claimedJob.GetId(),
					WorkerId: workerID,
					Outcome: &servicejobs.JobSettleFailure{Failure: &servicejobs.JobFailure{
						Error:     json.RawMessage(`{"message":"foo\njob failed but may be retried"}`),
						Retryable: true,
					}},
				},
				response: &servicejobs.JobSettleResponse{Job: &servicejobs.Job{
					Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusPending,
				}},
			},
			expect: &servicejobs.Job{
				Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusPending,
			},
		},
		{
			name: "Failure/Terminal",
			request: &core.JobExecuteRequest{
				Job: claimedJob, WorkerID: workerID, Deadline: time.Second,
			},
			handlerMock: &handlerMock{
				kind: "narrative.generate",
				handle: func(
					_ *testing.T, _ context.Context, _ *servicejobs.Job, _ core.ProviderCallRecorder,
				) (json.RawMessage, error) {
					return nil, errFoo
				},
			},
			settleMock: &settleMock{
				request: &servicejobs.JobSettleRequest{
					Id:       claimedJob.GetId(),
					WorkerId: workerID,
					Outcome: &servicejobs.JobSettleFailure{Failure: &servicejobs.JobFailure{
						Error: json.RawMessage(`{"message":"foo"}`),
					}},
				},
				response: &servicejobs.JobSettleResponse{Job: &servicejobs.Job{
					Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusFailed,
				}},
			},
			expect: &servicejobs.Job{
				Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusFailed,
			},
		},
		{
			name: "Failure/UnknownKind",
			request: &core.JobExecuteRequest{
				Job: unknownJob, WorkerID: workerID, Deadline: time.Second,
			},
			settleMock: &settleMock{
				request: &servicejobs.JobSettleRequest{
					Id:       unknownJob.GetId(),
					WorkerId: workerID,
					Outcome: &servicejobs.JobSettleFailure{Failure: &servicejobs.JobFailure{
						Error: json.RawMessage(`{"message":"job handler not found: \"unknown\""}`),
					}},
				},
				response: &servicejobs.JobSettleResponse{Job: &servicejobs.Job{
					Id: unknownJob.GetId(), Kind: unknownJob.GetKind(), Status: servicejobs.JobStatusFailed,
				}},
			},
			expect: &servicejobs.Job{
				Id: unknownJob.GetId(), Kind: unknownJob.GetKind(), Status: servicejobs.JobStatusFailed,
			},
		},
		{
			name: "Failure/Deadline",
			request: &core.JobExecuteRequest{
				Job: claimedJob, WorkerID: workerID, Deadline: time.Millisecond,
			},
			handlerMock: &handlerMock{
				kind: "narrative.generate",
				handle: func(
					t *testing.T, ctx context.Context, _ *servicejobs.Job, _ core.ProviderCallRecorder,
				) (json.RawMessage, error) {
					t.Helper()
					<-ctx.Done()
					require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)

					return nil, ctx.Err()
				},
			},
			settleMock: &settleMock{
				request: &servicejobs.JobSettleRequest{
					Id:       claimedJob.GetId(),
					WorkerId: workerID,
					Outcome: &servicejobs.JobSettleFailure{Failure: &servicejobs.JobFailure{
						Error: json.RawMessage(`{"message":"context deadline exceeded"}`),
					}},
				},
				response: &servicejobs.JobSettleResponse{Job: &servicejobs.Job{
					Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusFailed,
				}},
			},
			expect: &servicejobs.Job{
				Id: claimedJob.GetId(), Kind: claimedJob.GetKind(), Status: servicejobs.JobStatusFailed,
			},
		},
		{
			name: "Error/LostClaim",
			request: &core.JobExecuteRequest{
				Job: claimedJob, WorkerID: workerID, Deadline: time.Second,
			},
			handlerMock: &handlerMock{
				kind: "narrative.generate",
				handle: func(
					_ *testing.T, _ context.Context, _ *servicejobs.Job, _ core.ProviderCallRecorder,
				) (json.RawMessage, error) {
					return result, nil
				},
			},
			settleMock: &settleMock{
				request: &servicejobs.JobSettleRequest{
					Id:       claimedJob.GetId(),
					WorkerId: workerID,
					Outcome:  &servicejobs.JobSettleResult{Result: result},
				},
				err: errLostClaim,
			},
			expectLostClaimLog: true,
			expectErr:          errLostClaim,
		},
		{
			name: "Error/Settle",
			request: &core.JobExecuteRequest{
				Job: claimedJob, WorkerID: workerID, Deadline: time.Second,
			},
			handlerMock: &handlerMock{
				kind: "narrative.generate",
				handle: func(
					_ *testing.T, _ context.Context, _ *servicejobs.Job, _ core.ProviderCallRecorder,
				) (json.RawMessage, error) {
					return result, nil
				},
			},
			settleMock: &settleMock{
				request: &servicejobs.JobSettleRequest{
					Id:       claimedJob.GetId(),
					WorkerId: workerID,
					Outcome:  &servicejobs.JobSettleResult{Result: result},
				},
				err: errFoo,
			},
			expectErr: errFoo,
		},
		{
			name: "Error/SettleFailure",
			request: &core.JobExecuteRequest{
				Job: claimedJob, WorkerID: workerID, Deadline: time.Second,
			},
			handlerMock: &handlerMock{
				kind: "narrative.generate",
				handle: func(
					_ *testing.T, _ context.Context, _ *servicejobs.Job, _ core.ProviderCallRecorder,
				) (json.RawMessage, error) {
					return nil, errFoo
				},
			},
			settleMock: &settleMock{
				request: &servicejobs.JobSettleRequest{
					Id:       claimedJob.GetId(),
					WorkerId: workerID,
					Outcome: &servicejobs.JobSettleFailure{Failure: &servicejobs.JobFailure{
						Error: json.RawMessage(`{"message":"foo"}`),
					}},
				},
				err: errFoo,
			},
			expectErr: errFoo,
		},
		{
			name: "Error/MissingJob",
			request: &core.JobExecuteRequest{
				WorkerID: workerID, Deadline: time.Second,
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MissingWorkerID",
			request: &core.JobExecuteRequest{
				Job: claimedJob, Deadline: time.Second,
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MissingDeadline",
			request: &core.JobExecuteRequest{
				Job: claimedJob, WorkerID: workerID,
			},
			expectErr: core.ErrInvalidRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			jobsClient := jobsmocks.NewMockClient(t)
			logger := coremocks.NewMockJobExecuteLogger(t)

			var (
				handlers []core.JobHandler
				handler  *coremocks.MockJobHandler
			)
			if testCase.handlerMock != nil {
				handler = coremocks.NewMockJobHandler(t)
				handler.EXPECT().Kind().Return(testCase.handlerMock.kind).Once()
				handler.EXPECT().
					Handle(mock.Anything, testCase.request.Job, mock.Anything).
					RunAndReturn(func(
						ctx context.Context, job *servicejobs.Job, recorder core.ProviderCallRecorder,
					) (json.RawMessage, error) {
						return testCase.handlerMock.handle(t, ctx, job, recorder)
					}).
					Once()
				handlers = append(handlers, handler)
			}

			if testCase.recordProviderCallMock != nil {
				jobsClient.EXPECT().
					JobRecordProviderCall(mock.Anything, testCase.recordProviderCallMock.request).
					Return(
						testCase.recordProviderCallMock.response,
						testCase.recordProviderCallMock.err,
					).
					Once()
			}

			if testCase.settleMock != nil {
				jobsClient.EXPECT().
					JobSettle(mock.Anything, testCase.settleMock.request).
					Return(testCase.settleMock.response, testCase.settleMock.err).
					Once()
			}

			if testCase.expectLostClaimLog {
				logger.EXPECT().
					Err(mock.Anything, "job settle lost its claim", mock.Anything).
					Once()
			}

			service := core.NewJobExecute(jobsClient, logger, handlers...)
			response, err := service.Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, response)

			jobsClient.AssertExpectations(t)
			logger.AssertExpectations(t)

			if handler != nil {
				handler.AssertExpectations(t)
			}
		})
	}
}
