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

func TestPgLayerUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.LayerUpdateRequest

		expect    *dao.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.LayerUpdateRequest{
				ID: notesLayerID, OwnerID: ownerID, Revision: 1, Name: "Cast", Description: "Who speaks", Now: now,
			},

			expect: &dao.Layer{
				ID: notesLayerID, ProjectID: projectID, Position: 1, Kind: "notes", Name: "Cast",
				Description: "Who speaks", Revision: 2, UpdatedAt: now,
			},
		},
		{
			name: "Error/StaleRevision",

			request: &dao.LayerUpdateRequest{ID: notesLayerID, OwnerID: ownerID, Revision: 2, Name: "Cast", Now: now},

			expectErr: dao.ErrLayerUpdateNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: &dao.LayerUpdateRequest{ID: notesLayerID, OwnerID: otherOwnerID, Revision: 1, Name: "Cast", Now: now},

			expectErr: dao.ErrLayerUpdateNotFound,
		},
	}

	daoLayerUpdate := dao.NewPgLayerUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoLayerUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
