package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgLayersDelete(t *testing.T) {
	t.Parallel()

	// Two more layers, the first with a cell, let a delete leave a gap before a layer that has to move up
	// twice.
	places := &dao.Layer{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000023"), ProjectID: projectID, Position: 2,
		Kind: "notes", Name: "Places", UpdatedAt: updatedAt,
	}
	objects := &dao.Layer{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000024"), ProjectID: projectID, Position: 3,
		Kind: "notes", Name: "Objects", UpdatedAt: updatedAt,
	}
	placesCell := &dao.Cell{
		ProjectID: projectID, SceneID: sceneID1, LayerID: places.ID, Content: json.RawMessage(`{"text": "Paris"}`),
		UpdatedAt: updatedAt,
	}

	// layers and cells are the test project's rows before the delete, in order.
	fixtureLayers, fixtureCells := composerFixtures().layers, composerFixtures().cells
	layers := []*dao.Layer{fixtureLayers[0], fixtureLayers[1], places, objects}
	cells := []*dao.Cell{fixtureCells[0], fixtureCells[1], placesCell}

	testCases := []struct {
		name string

		request *dao.LayersDeleteRequest

		expect       []*dao.Layer
		expectLayers []*dao.Layer
		expectCells  []*dao.Cell
	}{
		{
			name: "Success",

			request: &dao.LayersDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{notesLayerID}},

			expect:       []*dao.Layer{layers[1]},
			expectLayers: []*dao.Layer{layers[0], layerAt(places, 1), layerAt(objects, 2)},
			expectCells:  []*dao.Cell{cells[0], placesCell},
		},
		{
			name: "Success/First",

			request: &dao.LayersDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{manuscriptLayerID}},

			expect:       []*dao.Layer{layers[0]},
			expectLayers: []*dao.Layer{layerAt(layers[1], 0), layerAt(places, 1), layerAt(objects, 2)},
			expectCells:  []*dao.Cell{cells[1], placesCell},
		},
		{
			name: "Success/NonAdjacent",

			request: &dao.LayersDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{manuscriptLayerID, places.ID}},

			expect:       []*dao.Layer{layers[0], places},
			expectLayers: []*dao.Layer{layerAt(layers[1], 0), layerAt(objects, 1)},
			expectCells:  []*dao.Cell{cells[1]},
		},
		{
			name: "Success/NotFound",

			request: &dao.LayersDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{absentID}},

			expectLayers: layers,
			expectCells:  cells,
		},
		{
			name: "Success/OtherProject",

			request: &dao.LayersDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{otherLayerID}},

			expectLayers: layers,
			expectCells:  cells,
		},
	}

	daoLayersDelete := dao.NewPgLayersDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				data.layers = append(data.layers, places, objects)
				data.cells = append(data.cells, placesCell)

				insertFixtures(ctx, t, data)

				result, err := daoLayersDelete.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.ElementsMatch(t, testCase.expect, result)

				// The layers after the deleted ones move up, and their cells go with them.
				require.Equal(t, testCase.expectLayers, listLayers(ctx, t, projectID))
				requireCellsEqual(t, testCase.expectCells, listCells(ctx, t, projectID))
			})
		})
	}
}
