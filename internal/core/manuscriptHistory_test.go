package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestManuscriptHistory(t *testing.T) {
	t.Parallel()

	request := &core.ManuscriptHistoryRequest{
		Actor:     core.Actor{UserID: ownerID},
		ProjectID: projectID,
	}
	entity := &dao.Manuscript{
		ID:        fixtureManuscriptID,
		ProjectID: projectID,
		Value:     staticManuscriptValue,
		CreatedAt: updatedAt,
	}
	expect := []*core.Manuscript{{
		ID:         fixtureManuscriptID,
		ProjectID:  projectID,
		Manuscript: staticManuscriptValue,
		CreatedAt:  updatedAt,
	}}
	errAccess := errors.New("access failure")
	errDAO := errors.New("dao failure")

	testCases := []struct {
		name string

		request *core.ManuscriptHistoryRequest
		setup   func(*coremocks.MockProjectAccessService, *coremocks.MockManuscriptHistoryDao)

		expect    []*core.Manuscript
		expectErr error
	}{
		{
			name:    "Success",
			request: request,
			setup: func(access *coremocks.MockProjectAccessService, history *coremocks.MockManuscriptHistoryDao) {
				access.EXPECT().Exec(mock.Anything, &core.ProjectAccessRequest{
					Actor: request.Actor, ProjectID: projectID,
				}).Return(projectFixture(), nil)
				history.EXPECT().Exec(mock.Anything, &dao.ManuscriptListRequest{
					ProjectID: projectID,
				}).Return([]*dao.Manuscript{entity}, nil)
			},
			expect: expect,
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.ManuscriptHistoryRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:    "Error/Access",
			request: request,
			setup: func(access *coremocks.MockProjectAccessService, _ *coremocks.MockManuscriptHistoryDao) {
				access.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil, errAccess)
			},
			expectErr: errAccess,
		},
		{
			name:    "Error/DAO",
			request: request,
			setup: func(access *coremocks.MockProjectAccessService, history *coremocks.MockManuscriptHistoryDao) {
				access.EXPECT().Exec(mock.Anything, mock.Anything).Return(projectFixture(), nil)
				history.EXPECT().Exec(mock.Anything, &dao.ManuscriptListRequest{
					ProjectID: projectID,
				}).Return(nil, errDAO)
			},
			expectErr: errDAO,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			access := coremocks.NewMockProjectAccessService(t)

			history := coremocks.NewMockManuscriptHistoryDao(t)
			if testCase.setup != nil {
				testCase.setup(access, history)
			}

			result, err := core.NewManuscriptHistory(access, history).Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
			access.AssertExpectations(t)
			history.AssertExpectations(t)
		})
	}
}
