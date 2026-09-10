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

func TestProjectGet(t *testing.T) {
	t.Parallel()

	request := &core.ProjectGetRequest{
		Actor:     core.Actor{UserID: ownerID},
		ProjectID: projectID,
	}
	ideaEntity := &dao.Idea{
		ProjectID:        projectID,
		VersionID:        fixtureIdeaVersionID,
		OwnerID:          ownerID,
		Title:            "The Answering Light",
		Genre:            "speculative",
		Seed:             "A foghorn answers from beneath the sea.",
		ProjectCreatedAt: createdAt,
		CreatedAt:        updatedAt,
	}
	stepEntity := &dao.StepValue{
		ID:        fixtureStepValueID,
		ProjectID: projectID,
		Key:       "outline",
		Value:     json.RawMessage(`{"unexpected":"still opaque"}`),
		CreatedAt: updatedAt,
	}
	manuscriptEntity := &dao.Manuscript{
		ID:        fixtureManuscriptID,
		ProjectID: projectID,
		Value:     staticManuscriptValue,
		CreatedAt: updatedAt,
	}
	expect := &core.ProjectSnapshot{
		ID:        projectID,
		CreatedAt: createdAt,
		Idea: &core.Idea{
			ProjectID:        projectID,
			VersionID:        fixtureIdeaVersionID,
			OwnerID:          ownerID,
			Title:            ideaEntity.Title,
			Genre:            ideaEntity.Genre,
			Seed:             ideaEntity.Seed,
			ProjectCreatedAt: createdAt,
			CreatedAt:        updatedAt,
		},
		StepValues: []*core.StepValue{{
			ID:        fixtureStepValueID,
			ProjectID: projectID,
			Key:       stepEntity.Key,
			Value:     stepEntity.Value,
			CreatedAt: updatedAt,
		}},
		Manuscript: &core.Manuscript{
			ID:         fixtureManuscriptID,
			ProjectID:  projectID,
			Manuscript: staticManuscriptValue,
			CreatedAt:  updatedAt,
		},
	}
	errAccess := errors.New("access failure")
	errIdea := errors.New("idea failure")
	errSteps := errors.New("step values failure")
	errManuscript := errors.New("manuscript failure")

	type mocks struct {
		access     *coremocks.MockProjectAccessService
		idea       *coremocks.MockProjectGetIdeaDao
		steps      *coremocks.MockProjectGetStepValueDao
		manuscript *coremocks.MockProjectGetManuscriptDao
	}

	testCases := []struct {
		name string

		request *core.ProjectGetRequest
		setup   func(mocks)

		expect    *core.ProjectSnapshot
		expectErr error
	}{
		{
			name:    "Success",
			request: request,
			setup: func(m mocks) {
				m.access.EXPECT().Exec(mock.Anything, &core.ProjectAccessRequest{
					Actor: request.Actor, ProjectID: projectID,
				}).Return(projectFixture(), nil)
				m.idea.EXPECT().Exec(mock.Anything, &dao.IdeaSelectRequest{
					ProjectID: projectID, OwnerID: ownerID,
				}).Return(ideaEntity, nil)
				m.steps.EXPECT().Exec(mock.Anything, &dao.StepValueCurrentListRequest{
					ProjectID: projectID,
				}).Return([]*dao.StepValue{stepEntity}, nil)
				m.manuscript.EXPECT().Exec(mock.Anything, &dao.ManuscriptSelectRequest{
					ProjectID: projectID,
				}).Return(manuscriptEntity, nil)
			},
			expect: expect,
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.ProjectGetRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:    "Error/Access",
			request: request,
			setup: func(m mocks) {
				m.access.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil, errAccess)
			},
			expectErr: errAccess,
		},
		{
			name:    "Error/Idea",
			request: request,
			setup: func(m mocks) {
				m.access.EXPECT().Exec(mock.Anything, mock.Anything).Return(projectFixture(), nil)
				m.idea.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil, errIdea)
			},
			expectErr: errIdea,
		},
		{
			name:    "Error/StepValues",
			request: request,
			setup: func(m mocks) {
				m.access.EXPECT().Exec(mock.Anything, mock.Anything).Return(projectFixture(), nil)
				m.idea.EXPECT().Exec(mock.Anything, mock.Anything).Return(ideaEntity, nil)
				m.steps.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil, errSteps)
			},
			expectErr: errSteps,
		},
		{
			name:    "Error/Manuscript",
			request: request,
			setup: func(m mocks) {
				m.access.EXPECT().Exec(mock.Anything, mock.Anything).Return(projectFixture(), nil)
				m.idea.EXPECT().Exec(mock.Anything, mock.Anything).Return(ideaEntity, nil)
				m.steps.EXPECT().Exec(mock.Anything, mock.Anything).Return([]*dao.StepValue{}, nil)
				m.manuscript.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil, errManuscript)
			},
			expectErr: errManuscript,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dependencies := mocks{
				access:     coremocks.NewMockProjectAccessService(t),
				idea:       coremocks.NewMockProjectGetIdeaDao(t),
				steps:      coremocks.NewMockProjectGetStepValueDao(t),
				manuscript: coremocks.NewMockProjectGetManuscriptDao(t),
			}
			if testCase.setup != nil {
				testCase.setup(dependencies)
			}

			result, err := core.NewProjectGet(
				dependencies.access,
				dependencies.idea,
				dependencies.steps,
				dependencies.manuscript,
			).Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
			dependencies.access.AssertExpectations(t)
			dependencies.idea.AssertExpectations(t)
			dependencies.steps.AssertExpectations(t)
			dependencies.manuscript.AssertExpectations(t)
		})
	}
}
