package dao_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgContentReads(t *testing.T) {
	t.Parallel()

	postgres.RunIsolatedTransactionalTest(
		t,
		configtest.PostgresPreset,
		migrations.Migrations,
		func(ctx context.Context, t *testing.T) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			createdAt := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
			projectID := uuid.NewSHA1(uuid.Nil, []byte("content-read-project"))

			_, err = db.NewInsert().Model(&dao.Project{
				ID:        projectID,
				OwnerID:   fixtureOwnerID,
				CreatedAt: createdAt,
			}).Exec(ctx)
			require.NoError(t, err)

			ideaVersions := make([]*dao.IdeaVersion, 0, 30)
			stepValues := make([]*dao.StepValue, 0, 31)
			manuscripts := make([]*dao.Manuscript, 0, 30)

			for index := range 30 {
				savedAt := createdAt.Add(time.Duration(index) * time.Second)
				ideaVersions = append(ideaVersions, &dao.IdeaVersion{
					ID:        uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("idea-%02d", index))),
					ProjectID: projectID,
					Title:     fmt.Sprintf("Title %02d", index),
					Genre:     "speculative",
					Seed:      fmt.Sprintf("Seed %02d", index),
					CreatedAt: savedAt,
				})
				stepValues = append(stepValues, &dao.StepValue{
					ID:        uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("step-%02d", index))),
					ProjectID: projectID,
					Key:       "outline",
					Value:     json.RawMessage(fmt.Sprintf(`{"version":%d}`, index)),
					CreatedAt: savedAt,
				})
				manuscripts = append(manuscripts, &dao.Manuscript{
					ID:        uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("manuscript-%02d", index))),
					ProjectID: projectID,
					Value:     json.RawMessage(fmt.Sprintf(`{"blocks":[{"version":%d}]}`, index)),
					CreatedAt: savedAt,
				})
			}

			stepValues = append(stepValues, &dao.StepValue{
				ID:        uuid.NewSHA1(uuid.Nil, []byte("step-characters")),
				ProjectID: projectID,
				Key:       "characters",
				Value:     json.RawMessage(`{"names":["Mara"]}`),
				CreatedAt: createdAt,
			})

			_, err = db.NewInsert().Model(&ideaVersions).Exec(ctx)
			require.NoError(t, err)
			_, err = db.NewInsert().Model(&stepValues).Exec(ctx)
			require.NoError(t, err)
			_, err = db.NewInsert().Model(&manuscripts).Exec(ctx)
			require.NoError(t, err)

			ideas, err := dao.NewPgIdeaVersionList().Exec(
				ctx,
				&dao.IdeaVersionListRequest{ProjectID: projectID},
			)
			require.NoError(t, err)
			require.Len(t, ideas, 25)
			require.Equal(t, "Title 29", ideas[0].Title)
			require.Equal(t, "Title 05", ideas[24].Title)

			outline, err := dao.NewPgStepValueList().Exec(
				ctx,
				&dao.StepValueListRequest{ProjectID: projectID, Key: "outline"},
			)
			require.NoError(t, err)
			require.Len(t, outline, 25)
			require.JSONEq(t, `{"version":29}`, string(outline[0].Value))

			currentSteps, err := dao.NewPgStepValueCurrentList().Exec(
				ctx,
				&dao.StepValueCurrentListRequest{ProjectID: projectID},
			)
			require.NoError(t, err)
			require.Len(t, currentSteps, 2)
			require.Equal(t, "characters", currentSteps[0].Key)
			require.Equal(t, "outline", currentSteps[1].Key)
			require.JSONEq(t, `{"version":29}`, string(currentSteps[1].Value))

			manuscriptHistory, err := dao.NewPgManuscriptList().Exec(
				ctx,
				&dao.ManuscriptListRequest{ProjectID: projectID},
			)
			require.NoError(t, err)
			require.Len(t, manuscriptHistory, 25)

			manuscript, err := dao.NewPgManuscriptSelect().Exec(
				ctx,
				&dao.ManuscriptSelectRequest{ProjectID: projectID},
			)
			require.NoError(t, err)
			require.JSONEq(t, `{"blocks":[{"version":29}]}`, string(manuscript.Value))
		},
	)
}
