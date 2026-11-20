package dao_test

import (
	"context"
	"testing"

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

		// held is the count of the bytes the rows of the test project hold, which the schema keeps.
		held    int64
		request *dao.ProjectBytesRequest

		expect    *dao.ProjectBytes
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectBytesRequest{ID: projectID},

			// The UTF-8 bytes of the project's own texts.
			expect: &dao.ProjectBytes{StoredBytes: 31},
		},
		{
			name: "Success/HeldRows",

			held:    64,
			request: &dao.ProjectBytesRequest{ID: projectID},

			// The bytes of the project's own texts (31), and of the rows it holds (64).
			expect: &dao.ProjectBytes{StoredBytes: 95},
		},
		{
			name: "Error/NotFound",

			request: &dao.ProjectBytesRequest{ID: absentID},

			expectErr: dao.ErrProjectBytesNotFound,
		},
	}

	daoProjectBytes := dao.NewPgProjectBytes()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, projectFixtures())

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				_, err = db.NewRaw("UPDATE projects SET stored_bytes = ?0 WHERE id = ?1", testCase.held, projectID).Exec(ctx)
				require.NoError(t, err)

				result, err := daoProjectBytes.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
