package core_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestStepValueHistory(t *testing.T) {
	t.Parallel()

	request := &core.StepValueHistoryRequest{
		Actor:     core.Actor{UserID: ownerID},
		ProjectID: projectID,
		Key:       "outline",
	}
	value := json.RawMessage(`{"formerSchema":"intentionally invalid"}`)
	entity := &dao.StepValue{
		ID:        fixtureStepValueID,
		ProjectID: projectID,
		Key:       request.Key,
		Value:     value,
		CreatedAt: updatedAt,
	}
	expect := []*core.StepValue{{
		ID:        fixtureStepValueID,
		ProjectID: projectID,
		Key:       request.Key,
		Value:     value,
		CreatedAt: updatedAt,
	}}
	errAccess := errors.New("access failure")
	errDAO := errors.New("dao failure")

	testCases := []struct {
		name string

		request *core.StepValueHistoryRequest
		setup   func(*coremocks.MockProjectAccessService, *coremocks.MockStepValueHistoryDao)

		expect    []*core.StepValue
		expectErr error
	}{
		{
			name:    "Success",
			request: request,
			setup: func(access *coremocks.MockProjectAccessService, history *coremocks.MockStepValueHistoryDao) {
				access.EXPECT().Exec(mock.Anything, &core.ProjectAccessRequest{
					Actor: request.Actor, ProjectID: projectID,
				}).Return(projectFixture(), nil)
				history.EXPECT().Exec(mock.Anything, &dao.StepValueListRequest{
					ProjectID: projectID, Key: request.Key,
				}).Return([]*dao.StepValue{entity}, nil)
			},
			expect: expect,
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.StepValueHistoryRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:    "Error/Access",
			request: request,
			setup: func(access *coremocks.MockProjectAccessService, _ *coremocks.MockStepValueHistoryDao) {
				access.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil, errAccess)
			},
			expectErr: errAccess,
		},
		{
			name:    "Error/DAO",
			request: request,
			setup: func(access *coremocks.MockProjectAccessService, history *coremocks.MockStepValueHistoryDao) {
				access.EXPECT().Exec(mock.Anything, mock.Anything).Return(projectFixture(), nil)
				history.EXPECT().Exec(mock.Anything, &dao.StepValueListRequest{
					ProjectID: projectID, Key: request.Key,
				}).Return(nil, errDAO)
			},
			expectErr: errDAO,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			access := coremocks.NewMockProjectAccessService(t)

			history := coremocks.NewMockStepValueHistoryDao(t)
			if testCase.setup != nil {
				testCase.setup(access, history)
			}

			result, err := core.NewStepValueHistory(access, history).Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
			access.AssertExpectations(t)
			history.AssertExpectations(t)
		})
	}
}
