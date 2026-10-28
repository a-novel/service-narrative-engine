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

func TestPgLayersUpdate(t *testing.T) {
	t.Parallel()

	layers := composerFixtures().layers

	// updateManuscript and updateNotes rewrite the project's two layers, which become manuscript and notes.
	updateManuscript := dao.LayersUpdateItem{ID: manuscriptLayerID, Name: "Draft", Description: "What gets written"}
	updateNotes := dao.LayersUpdateItem{ID: notesLayerID, Name: "Cast"}
	manuscript := &dao.Layer{
		ID: manuscriptLayerID, ProjectID: projectID, Position: 0, Kind: "manuscript", Name: "Draft",
		Description: "What gets written", UpdatedAt: now,
	}
	notes := &dao.Layer{ID: notesLayerID, ProjectID: projectID, Position: 1, Kind: "notes", Name: "Cast", UpdatedAt: now}

	testCases := []struct {
		name string

		request *dao.LayersUpdateRequest

		expect []*dao.Layer
		// expectLayers is the project's order after the update.
		expectLayers []*dao.Layer
	}{
		{
			name: "Success",

			request: &dao.LayersUpdateRequest{ProjectID: projectID, Layers: []dao.LayersUpdateItem{updateNotes}, Now: now},

			expect:       []*dao.Layer{notes},
			expectLayers: []*dao.Layer{layers[0], notes},
		},
		{
			name: "Success/Batch",

			request: &dao.LayersUpdateRequest{
				ProjectID: projectID, Layers: []dao.LayersUpdateItem{updateManuscript, updateNotes}, Now: now,
			},

			expect:       []*dao.Layer{manuscript, notes},
			expectLayers: []*dao.Layer{manuscript, notes},
		},
		{
			name: "Success/NotFound",

			request: &dao.LayersUpdateRequest{
				ProjectID: projectID, Layers: []dao.LayersUpdateItem{updateNotes, {ID: absentID, Name: "Places"}}, Now: now,
			},

			expectLayers: layers[:2],
		},
		{
			name: "Success/OtherProject",

			request: &dao.LayersUpdateRequest{
				ProjectID: projectID, Layers: []dao.LayersUpdateItem{updateNotes, {ID: otherLayerID, Name: "Places"}}, Now: now,
			},

			expectLayers: layers[:2],
		},
	}

	daoLayersUpdate := dao.NewPgLayersUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoLayersUpdate.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.ElementsMatch(t, testCase.expect, result)

				// An ID the project does not hold leaves every layer unchanged, the valid ones included.
				require.Equal(t, testCase.expectLayers, listLayers(ctx, t, projectID))
			})
		})
	}
}
