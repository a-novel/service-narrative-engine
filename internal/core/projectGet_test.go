package core_test

import (
	"encoding/json"
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

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

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

		projectAccess := coremocks.NewMockProjectAccessService(t)
		projectAccess.EXPECT().
			Exec(mock.Anything, &core.ProjectAccessRequest{
				Actor:     request.Actor,
				ProjectID: projectID,
			}).
			Return(projectFixture(), nil)

		ideaDao := coremocks.NewMockProjectGetIdeaDao(t)
		ideaDao.EXPECT().
			Exec(mock.Anything, &dao.IdeaSelectRequest{ProjectID: projectID, OwnerID: ownerID}).
			Return(ideaEntity, nil)

		stepValueDao := coremocks.NewMockProjectGetStepValueDao(t)
		stepValueDao.EXPECT().
			Exec(mock.Anything, &dao.StepValueCurrentListRequest{ProjectID: projectID}).
			Return([]*dao.StepValue{stepEntity}, nil)

		manuscriptDao := coremocks.NewMockProjectGetManuscriptDao(t)
		manuscriptDao.EXPECT().
			Exec(mock.Anything, &dao.ManuscriptSelectRequest{ProjectID: projectID}).
			Return(manuscriptEntity, nil)

		result, err := core.NewProjectGet(
			projectAccess,
			ideaDao,
			stepValueDao,
			manuscriptDao,
		).Exec(t.Context(), request)

		require.NoError(t, err)
		require.Equal(t, projectID, result.ID)
		require.Equal(t, fixtureIdeaVersionID, result.Idea.VersionID)
		require.Equal(t, stepEntity.Value, result.StepValues[0].Value)
		require.Equal(t, manuscriptEntity.Value, result.Manuscript.Manuscript)
	})

	t.Run("Error/OtherOwnerStopsBeforeContentReads", func(t *testing.T) {
		t.Parallel()

		projectAccess := coremocks.NewMockProjectAccessService(t)
		projectAccess.EXPECT().
			Exec(mock.Anything, &core.ProjectAccessRequest{
				Actor:     request.Actor,
				ProjectID: projectID,
			}).
			Return(nil, core.ErrProjectNotFound)

		result, err := core.NewProjectGet(
			projectAccess,
			coremocks.NewMockProjectGetIdeaDao(t),
			coremocks.NewMockProjectGetStepValueDao(t),
			coremocks.NewMockProjectGetManuscriptDao(t),
		).Exec(t.Context(), request)

		require.ErrorIs(t, err, core.ErrProjectNotFound)
		require.Nil(t, result)
	})
}
