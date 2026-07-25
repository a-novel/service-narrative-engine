package core_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
)

func TestJobGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")
	jobID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	job := &core.Job{
		Id:      jobID.String(),
		Kind:    "narrative.generate",
		OwnerId: testActor.UserID.String(),
		Status:  core.JobStatusSucceeded,
	}

	type serviceMock struct {
		response *servicejobs.JobGetResponse
		err      error
	}

	testCases := []struct {
		name string

		request *core.JobGetRequest

		serviceMock *serviceMock

		expect    *core.Job
		expectErr error
	}{
		{
			name:        "Success",
			request:     &core.JobGetRequest{ID: jobID, Actor: testActor},
			serviceMock: &serviceMock{response: &servicejobs.JobGetResponse{Job: job}},
			expect:      job,
		},
		{
			name:    "Error/OtherActorNotFound",
			request: &core.JobGetRequest{ID: jobID, Actor: testActor},
			serviceMock: &serviceMock{
				err: status.Error(codes.NotFound, "job not found"),
			},
			expectErr: core.ErrJobGetNotFound,
		},
		{
			name:      "Error/MissingID",
			request:   &core.JobGetRequest{Actor: testActor},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "Error/MissingActor",
			request:   &core.JobGetRequest{ID: jobID},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:        "Error/Service",
			request:     &core.JobGetRequest{ID: jobID, Actor: testActor},
			serviceMock: &serviceMock{err: errFoo},
			expectErr:   errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			queue := coremocks.NewMockJobGetService(t)
			if testCase.serviceMock != nil {
				queue.EXPECT().
					JobGet(mock.Anything, &servicejobs.JobGetRequest{
						Id:      testCase.request.ID.String(),
						OwnerId: testCase.request.Actor.UserID.String(),
					}).
					Return(testCase.serviceMock.response, testCase.serviceMock.err).
					Once()
			}

			service := core.NewJobGet(queue)
			response, err := service.Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, response)
			queue.AssertExpectations(t)
		})
	}
}
