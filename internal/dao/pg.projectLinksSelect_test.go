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

func TestPgProjectLinksSelect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.ProjectLinksSelectRequest

		expect    *dao.ProjectLinks
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectLinksSelectRequest{ProjectID: projectID},

			expect: instanceFixtures().links[0],
		},
		{
			// The project of the other owner never wrote its links.
			name: "Error/NotFound",

			request: &dao.ProjectLinksSelectRequest{ProjectID: otherProjectID},

			expectErr: dao.ErrProjectLinksNotFound,
		},
		{
			name: "Error/NotFound/NoProject",

			request: &dao.ProjectLinksSelectRequest{ProjectID: absentID},

			expectErr: dao.ErrProjectLinksNotFound,
		},
	}

	daoProjectLinksSelect := dao.NewPgProjectLinksSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, instanceFixtures())

				result, err := daoProjectLinksSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
