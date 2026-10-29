package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

// pgLockNotAvailable is the SQLSTATE PostgreSQL raises when a statement gives up a lock wait.
const pgLockNotAvailable = "55P03"

func TestPgSceneLock(t *testing.T) {
	t.Parallel()

	// A first write locks the first scene and writes one of its cells. A second write then locks the scenes
	// of the case in a transaction of its own, which gives up any lock wait after 100ms.
	testCases := []struct {
		name string

		request *dao.SceneLockRequest

		expectWait bool
	}{
		{
			name: "Success/OtherScene",

			request: &dao.SceneLockRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID2}},
		},
		{
			// The other project holds no such scene, so the lock skips it.
			name: "Success/OtherProject",

			request: &dao.SceneLockRequest{ProjectID: otherProjectID, IDs: []uuid.UUID{sceneID1}},
		},
		{
			name: "Success/WaitsForScene",

			request: &dao.SceneLockRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID1}},

			expectWait: true,
		},
		{
			name: "Success/WaitsForSharedScene",

			request: &dao.SceneLockRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID2, sceneID1}},

			expectWait: true,
		},
	}

	daoSceneLock := dao.NewPgSceneLock()
	daoCellUpsert := dao.NewPgCellUpsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				err := postgres.WithinTx(ctx, nil, func(firstCtx context.Context) error {
					err := daoSceneLock.Exec(firstCtx, &dao.SceneLockRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID1}})
					require.NoError(t, err)

					_, err = daoCellUpsert.Exec(firstCtx, &dao.CellUpsertRequest{
						ProjectID: projectID, SceneID: sceneID1, LayerID: notesLayerID,
						Content: json.RawMessage(`{"text": "Paul"}`), Now: now,
					})
					require.NoError(t, err)

					return postgres.WithinTx(ctx, nil, func(secondCtx context.Context) error {
						db, err := postgres.GetContext(secondCtx)
						require.NoError(t, err)

						_, err = db.NewRaw("SET LOCAL lock_timeout = '100ms'").Exec(secondCtx)
						require.NoError(t, err)

						return daoSceneLock.Exec(secondCtx, testCase.request)
					})
				})

				if !testCase.expectWait {
					require.NoError(t, err)

					return
				}

				var pgErr pgdriver.Error

				require.ErrorAs(t, err, &pgErr)
				require.Equal(t, pgLockNotAvailable, pgErr.Field('C'))
			})
		})
	}
}
