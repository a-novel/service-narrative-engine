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

func TestIdeaHistory(t *testing.T) {
	t.Parallel()

	request := &core.IdeaHistoryRequest{
		Actor:     core.Actor{UserID: ownerID},
		ProjectID: projectID,
	}
	entity := &dao.IdeaVersion{
		ID:        fixtureIdeaVersionID,
		ProjectID: projectID,
		Title:     "The Answering Light",
		Genre:     "speculative",
		Seed:      "A foghorn answers from beneath the sea.",
		CreatedAt: updatedAt,
	}
	expect := []*core.Idea{{
		ProjectID:        projectID,
		VersionID:        fixtureIdeaVersionID,
		OwnerID:          ownerID,
		Title:            entity.Title,
		Genre:            entity.Genre,
		Seed:             entity.Seed,
		ProjectCreatedAt: createdAt,
		CreatedAt:        updatedAt,
	}}
	errAccess := errors.New("access failure")
	errDAO := errors.New("dao failure")

	testCases := []struct {
		name string

		request *core.IdeaHistoryRequest
		setup   func(*coremocks.MockProjectAccessService, *coremocks.MockIdeaHistoryDao)

		expect    []*core.Idea
		expectErr error
	}{
		{
			name:    "Success",
			request: request,
			setup: func(access *coremocks.MockProjectAccessService, history *coremocks.MockIdeaHistoryDao) {
				access.EXPECT().Exec(mock.Anything, &core.ProjectAccessRequest{
					Actor: request.Actor, ProjectID: projectID,
				}).Return(projectFixture(), nil)
				history.EXPECT().Exec(mock.Anything, &dao.IdeaVersionListRequest{
					ProjectID: projectID,
				}).Return([]*dao.IdeaVersion{entity}, nil)
			},
			expect: expect,
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.IdeaHistoryRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:    "Error/Access",
			request: request,
			setup: func(access *coremocks.MockProjectAccessService, _ *coremocks.MockIdeaHistoryDao) {
				access.EXPECT().Exec(mock.Anything, &core.ProjectAccessRequest{
					Actor: request.Actor, ProjectID: projectID,
				}).Return(nil, errAccess)
			},
			expectErr: errAccess,
		},
		{
			name:    "Error/DAO",
			request: request,
			setup: func(access *coremocks.MockProjectAccessService, history *coremocks.MockIdeaHistoryDao) {
				access.EXPECT().Exec(mock.Anything, mock.Anything).Return(projectFixture(), nil)
				history.EXPECT().Exec(mock.Anything, &dao.IdeaVersionListRequest{
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

			history := coremocks.NewMockIdeaHistoryDao(t)
			if testCase.setup != nil {
				testCase.setup(access, history)
			}

			result, err := core.NewIdeaHistory(access, history).Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
			access.AssertExpectations(t)
			history.AssertExpectations(t)
		})
	}
}
