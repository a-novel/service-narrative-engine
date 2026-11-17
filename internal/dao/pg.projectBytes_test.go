package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgProjectBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		// write runs before the read, in its transaction, as the budget check reads the count after a write.
		write   *dao.BeatsDeleteRequest
		request *dao.ProjectBytesRequest

		expect    *dao.ProjectBytes
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectBytesRequest{ID: projectID},

			// The UTF-8 bytes of the project's own texts (31), of its beats (33), scenes (34) and layers (31),
			// and of its cells' documents as text (31).
			expect: &dao.ProjectBytes{StoredBytes: 160},
		},
		{
			name: "Success/AfterWrite",

			write:   &dao.BeatsDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{beatID1}},
			request: &dao.ProjectBytesRequest{ID: projectID},

			// The delete takes the first beat (9), the three scenes it holds (25) and the cells of its first
			// scene (31), which its cascade removes.
			expect: &dao.ProjectBytes{StoredBytes: 95},
		},
		{
			name: "Error/NotFound",

			request: &dao.ProjectBytesRequest{ID: absentID},

			expectErr: dao.ErrProjectBytesNotFound,
		},
	}

	daoBeatsDelete := dao.NewPgBeatsDelete()
	daoProjectBytes := dao.NewPgProjectBytes()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				err := postgres.WithinTx(ctx, nil, func(ctx context.Context) error {
					if testCase.write != nil {
						_, err := daoBeatsDelete.Exec(ctx, testCase.write)
						require.NoError(t, err)
					}

					result, err := daoProjectBytes.Exec(ctx, testCase.request)
					require.ErrorIs(t, err, testCase.expectErr)
					require.Equal(t, testCase.expect, result)

					return nil
				})
				require.NoError(t, err)
			})
		})
	}
}
