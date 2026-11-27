package dao_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgProjectBytes(t *testing.T) {
	t.Parallel()

	// held sums the bytes of the rows the test project holds in the instance fixtures: the texts and the
	// documents of its two instances, and its links. PostgreSQL counts a JSON document as it spells it.
	held := len("scenes-composer"+"1.0.0"+"Composer"+instanceContent) +
		len("characters"+"0.1.0-dev.1"+"Characters"+instanceContentNext) +
		len(projectLinks)

	testCases := []struct {
		name string

		fixtures fixtures
		request  *dao.ProjectBytesRequest

		expect    *dao.ProjectBytes
		expectErr error
	}{
		{
			name: "Success",

			fixtures: projectFixtures(),
			request:  &dao.ProjectBytesRequest{ID: projectID},

			// The UTF-8 bytes of the project's own texts.
			expect: &dao.ProjectBytes{StoredBytes: 31},
		},
		{
			name: "Success/HeldRows",

			fixtures: instanceFixtures(),
			request:  &dao.ProjectBytesRequest{ID: projectID},

			// The bytes of the project's own texts, and of the rows it holds.
			expect: &dao.ProjectBytes{StoredBytes: int64(31 + held)},
		},
		{
			name: "Error/NotFound",

			fixtures: projectFixtures(),
			request:  &dao.ProjectBytesRequest{ID: absentID},

			expectErr: dao.ErrProjectNotFound,
		},
	}

	daoProjectBytes := dao.NewPgProjectBytes()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, testCase.fixtures)

				result, err := daoProjectBytes.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				requireStoredBytes(ctx, t)
			})
		})
	}
}
