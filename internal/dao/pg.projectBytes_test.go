package dao_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgProjectBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

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

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoProjectBytes.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
