package core_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
)

func TestJobEnqueue(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")
	createdJob := &core.Job{
		Id:          "00000000-0000-0000-0000-000000000001",
		Kind:        "narrative.generate",
		OwnerId:     testActor.UserID.String(),
		Status:      core.JobStatusPending,
		MaxAttempts: 1,
	}
	attachedJob := &core.Job{
		Id:          "00000000-0000-0000-0000-000000000002",
		Kind:        "narrative.generate",
		OwnerId:     testActor.UserID.String(),
		Status:      core.JobStatusClaimed,
		MaxAttempts: 2,
	}
	idempotencyKey := "manuscript-42"

	type serviceMock struct {
		request  *servicejobs.JobEnqueueRequest
		response *servicejobs.JobEnqueueResponse
		err      error
	}

	testCases := []struct {
		name string

		request *core.JobEnqueueRequest

		serviceMock *serviceMock

		expect    *core.JobEnqueueResponse
		expectErr error
	}{
		{
			name: "Success/Created",
			request: &core.JobEnqueueRequest{
				Kind:           "narrative.generate",
				Payload:        json.RawMessage(`{"owner_id":"00000000-0000-0000-0000-000000000099"}`),
				Actor:          testActor,
				IdempotencyKey: idempotencyKey,
			},
			serviceMock: &serviceMock{
				request: &servicejobs.JobEnqueueRequest{
					Kind:           "narrative.generate",
					Payload:        json.RawMessage(`{"owner_id":"00000000-0000-0000-0000-000000000099"}`),
					OwnerId:        testActor.UserID.String(),
					IdempotencyKey: &idempotencyKey,
					MaxAttempts:    1,
				},
				response: &servicejobs.JobEnqueueResponse{Job: createdJob, Created: true},
			},
			expect: &core.JobEnqueueResponse{Job: createdJob, Created: true},
		},
		{
			name: "Success/Attached",
			request: &core.JobEnqueueRequest{
				Kind:           "narrative.generate",
				Payload:        json.RawMessage(`{"idea_id":"00000000-0000-0000-0000-000000000043"}`),
				Actor:          testActor,
				IdempotencyKey: idempotencyKey,
				MaxAttempts:    2,
			},
			serviceMock: &serviceMock{
				request: &servicejobs.JobEnqueueRequest{
					Kind:           "narrative.generate",
					Payload:        json.RawMessage(`{"idea_id":"00000000-0000-0000-0000-000000000043"}`),
					OwnerId:        testActor.UserID.String(),
					IdempotencyKey: &idempotencyKey,
					MaxAttempts:    2,
				},
				response: &servicejobs.JobEnqueueResponse{Job: attachedJob, Created: false},
			},
			expect: &core.JobEnqueueResponse{Job: attachedJob, Created: false},
		},
		{
			name: "Error/MissingActor",
			request: &core.JobEnqueueRequest{
				Kind:    "narrative.generate",
				Payload: json.RawMessage(`{}`),
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/BlankKind",
			request: &core.JobEnqueueRequest{
				Kind:    "   ",
				Payload: json.RawMessage(`{}`),
				Actor:   testActor,
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MissingPayload",
			request: &core.JobEnqueueRequest{
				Kind:  "narrative.generate",
				Actor: testActor,
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MaxAttempts",
			request: &core.JobEnqueueRequest{
				Kind:        "narrative.generate",
				Payload:     json.RawMessage(`{}`),
				Actor:       testActor,
				MaxAttempts: 33,
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Service",
			request: &core.JobEnqueueRequest{
				Kind:        "narrative.generate",
				Payload:     json.RawMessage(`{}`),
				Actor:       testActor,
				MaxAttempts: 1,
			},
			serviceMock: &serviceMock{
				request: &servicejobs.JobEnqueueRequest{
					Kind:        "narrative.generate",
					Payload:     json.RawMessage(`{}`),
					OwnerId:     testActor.UserID.String(),
					MaxAttempts: 1,
				},
				err: errFoo,
			},
			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			queue := coremocks.NewMockJobEnqueueService(t)
			if testCase.serviceMock != nil {
				queue.EXPECT().
					JobEnqueue(mock.Anything, testCase.serviceMock.request).
					Return(testCase.serviceMock.response, testCase.serviceMock.err).
					Once()
			}

			service := core.NewJobEnqueue(queue)
			response, err := service.Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, response)
			queue.AssertExpectations(t)
		})
	}
}
