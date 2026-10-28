package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgLayerKind(t *testing.T) {
	t.Parallel()

	// A second manuscript layer in the test project, so the manuscript kind counts two layers there.
	draft := &dao.Layer{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000023"), ProjectID: projectID, Position: 2,
		Kind: "manuscript", Name: "Draft", UpdatedAt: updatedAt,
	}

	testCases := []struct {
		name string

		request *dao.LayerKindRequest

		expect []*dao.LayerKind
	}{
		{
			name: "Success",

			request: &dao.LayerKindRequest{ProjectID: projectID, IDs: []uuid.UUID{manuscriptLayerID, notesLayerID}},

			// The draft counts though it is not requested, and the other project's manuscript does not.
			expect: []*dao.LayerKind{
				{ID: manuscriptLayerID, Kind: "manuscript", Count: 2},
				{ID: notesLayerID, Kind: "notes", Count: 1},
			},
		},
		{
			name: "Success/SharedKind",

			request: &dao.LayerKindRequest{ProjectID: projectID, IDs: []uuid.UUID{manuscriptLayerID, draft.ID}},

			expect: []*dao.LayerKind{
				{ID: manuscriptLayerID, Kind: "manuscript", Count: 2},
				{ID: draft.ID, Kind: "manuscript", Count: 2},
			},
		},
		{
			name: "Success/NotFound",

			request: &dao.LayerKindRequest{ProjectID: projectID, IDs: []uuid.UUID{absentID}},
		},
		{
			name: "Success/OtherProject",

			request: &dao.LayerKindRequest{ProjectID: projectID, IDs: []uuid.UUID{otherLayerID}},
		},
	}

	daoLayerKind := dao.NewPgLayerKind()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				data.layers = append(data.layers, draft)

				insertFixtures(ctx, t, data)

				result, err := daoLayerKind.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.ElementsMatch(t, testCase.expect, result)
			})
		})
	}
}
