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

func TestContentHistory(t *testing.T) {
	t.Parallel()

	t.Run("Idea", func(t *testing.T) {
		t.Parallel()

		projectAccess := coremocks.NewMockProjectAccessService(t)
		projectAccess.EXPECT().
			Exec(mock.Anything, &core.ProjectAccessRequest{
				Actor:     core.Actor{UserID: ownerID},
				ProjectID: projectID,
			}).
			Return(projectFixture(), nil)

		historyDao := coremocks.NewMockIdeaHistoryDao(t)
		historyDao.EXPECT().
			Exec(mock.Anything, &dao.IdeaVersionListRequest{ProjectID: projectID}).
			Return([]*dao.IdeaVersion{{
				ID:        fixtureIdeaVersionID,
				ProjectID: projectID,
				Title:     "The Answering Light",
				Genre:     "speculative",
				Seed:      "A foghorn answers from beneath the sea.",
				CreatedAt: updatedAt,
			}}, nil)

		result, err := core.NewIdeaHistory(projectAccess, historyDao).Exec(
			t.Context(),
			&core.IdeaHistoryRequest{
				Actor:     core.Actor{UserID: ownerID},
				ProjectID: projectID,
			},
		)

		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Equal(t, fixtureIdeaVersionID, result[0].VersionID)
		require.Equal(t, ownerID, result[0].OwnerID)
	})

	t.Run("StepValue", func(t *testing.T) {
		t.Parallel()

		projectAccess := coremocks.NewMockProjectAccessService(t)
		projectAccess.EXPECT().
			Exec(mock.Anything, &core.ProjectAccessRequest{
				Actor:     core.Actor{UserID: ownerID},
				ProjectID: projectID,
			}).
			Return(projectFixture(), nil)

		value := json.RawMessage(`{"formerSchema":"intentionally invalid"}`)
		historyDao := coremocks.NewMockStepValueHistoryDao(t)
		historyDao.EXPECT().
			Exec(mock.Anything, &dao.StepValueListRequest{ProjectID: projectID, Key: "outline"}).
			Return([]*dao.StepValue{{
				ID:        fixtureStepValueID,
				ProjectID: projectID,
				Key:       "outline",
				Value:     value,
				CreatedAt: updatedAt,
			}}, nil)

		result, err := core.NewStepValueHistory(projectAccess, historyDao).Exec(
			t.Context(),
			&core.StepValueHistoryRequest{
				Actor:     core.Actor{UserID: ownerID},
				ProjectID: projectID,
				Key:       "outline",
			},
		)

		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Equal(t, value, result[0].Value)
	})

	t.Run("Manuscript", func(t *testing.T) {
		t.Parallel()

		projectAccess := coremocks.NewMockProjectAccessService(t)
		projectAccess.EXPECT().
			Exec(mock.Anything, &core.ProjectAccessRequest{
				Actor:     core.Actor{UserID: ownerID},
				ProjectID: projectID,
			}).
			Return(projectFixture(), nil)

		historyDao := coremocks.NewMockManuscriptHistoryDao(t)
		historyDao.EXPECT().
			Exec(mock.Anything, &dao.ManuscriptListRequest{ProjectID: projectID}).
			Return([]*dao.Manuscript{{
				ID:        fixtureManuscriptID,
				ProjectID: projectID,
				Value:     staticManuscriptValue,
				CreatedAt: updatedAt,
			}}, nil)

		result, err := core.NewManuscriptHistory(projectAccess, historyDao).Exec(
			t.Context(),
			&core.ManuscriptHistoryRequest{
				Actor:     core.Actor{UserID: ownerID},
				ProjectID: projectID,
			},
		)

		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Equal(t, staticManuscriptValue, result[0].Manuscript)
	})

	t.Run("OtherOwnerStopsBeforeHistoryRead", func(t *testing.T) {
		t.Parallel()

		projectAccess := coremocks.NewMockProjectAccessService(t)
		projectAccess.EXPECT().
			Exec(mock.Anything, &core.ProjectAccessRequest{
				Actor:     core.Actor{UserID: ownerID},
				ProjectID: projectID,
			}).
			Return(nil, core.ErrProjectNotFound)

		result, err := core.NewStepValueHistory(
			projectAccess,
			coremocks.NewMockStepValueHistoryDao(t),
		).Exec(t.Context(), &core.StepValueHistoryRequest{
			Actor:     core.Actor{UserID: ownerID},
			ProjectID: projectID,
			Key:       "outline",
		})

		require.ErrorIs(t, err, core.ErrProjectNotFound)
		require.Nil(t, result)
	})
}
