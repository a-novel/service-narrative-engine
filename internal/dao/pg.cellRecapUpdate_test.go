package dao_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgCellRecapUpdate(t *testing.T) {
	t.Parallel()

	recap := &dao.Recap{
		Condensed: "Marie attend.", Compact: "Marie attend.", Essentials: "Marie.", Names: []string{"Marie"},
	}
	recapJSON, err := json.Marshal(recap)
	require.NoError(t, err)

	older := &dao.Recap{Condensed: "Avant.", Compact: "Avant.", Essentials: "Avant.", Names: []string{}}

	// recapped returns a copy of a cell holding a recap that read the cell at stamp.
	recapped := func(cell *dao.Cell, recap *dao.Recap, stamp time.Time) *dao.Cell {
		copied := *cell
		copied.Recap, copied.RecapCellUpdatedAt = recap, &stamp

		return &copied
	}

	// otherCell is a cell of the other project, which no write of the test project may change.
	otherCell := &dao.Cell{
		ProjectID: otherProjectID, SceneID: otherSceneID, LayerID: otherLayerID,
		Content: json.RawMessage(`{"blocks": []}`), UpdatedAt: updatedAt,
	}

	request := func(sceneID, layerID uuid.UUID, stamp time.Time) *dao.CellRecapUpdateRequest {
		return &dao.CellRecapUpdateRequest{
			ProjectID: projectID, SceneID: sceneID, LayerID: layerID, Recap: recapJSON, CellUpdatedAt: stamp,
		}
	}

	testCases := []struct {
		name string

		// stored is the recap the manuscript cell holds before the write, read at createdAt or updatedAt.
		stored   *dao.Recap
		storedAt time.Time

		request *dao.CellRecapUpdateRequest

		// expectCell is the manuscript cell after the write. The notes cell never changes.
		expectCell func(cell *dao.Cell) *dao.Cell
		expectErr  error
	}{
		{
			name: "Success",

			request: request(sceneID1, manuscriptLayerID, updatedAt),

			expectCell: func(cell *dao.Cell) *dao.Cell { return recapped(cell, recap, updatedAt) },
		},
		{
			// The cell changed after the consolidation read it: the recap is stored, and stale.
			name: "Success/CellChangedSince",

			request: request(sceneID1, manuscriptLayerID, createdAt),

			expectCell: func(cell *dao.Cell) *dao.Cell { return recapped(cell, recap, createdAt) },
		},
		{
			name: "Success/ReplacesOlder",

			stored: older, storedAt: createdAt,

			request: request(sceneID1, manuscriptLayerID, updatedAt),

			expectCell: func(cell *dao.Cell) *dao.Cell { return recapped(cell, recap, updatedAt) },
		},
		{
			// Another result of the same consolidation replaces the first.
			name: "Success/SameStamp",

			stored: older, storedAt: updatedAt,

			request: request(sceneID1, manuscriptLayerID, updatedAt),

			expectCell: func(cell *dao.Cell) *dao.Cell { return recapped(cell, recap, updatedAt) },
		},
		{
			name: "Error/OlderThanStored",

			stored: older, storedAt: updatedAt,

			request: request(sceneID1, manuscriptLayerID, createdAt),

			expectCell: func(cell *dao.Cell) *dao.Cell { return recapped(cell, older, updatedAt) },
			expectErr:  dao.ErrCellRecapUpdateConflict,
		},
		{
			name: "Error/NewerThanCell",

			request: request(sceneID1, manuscriptLayerID, now),

			expectCell: func(cell *dao.Cell) *dao.Cell { return cell },
			expectErr:  dao.ErrCellRecapUpdateConflict,
		},
		{
			name: "Error/NoCell",

			request: request(sceneID2, manuscriptLayerID, updatedAt),

			expectCell: func(cell *dao.Cell) *dao.Cell { return cell },
			expectErr:  dao.ErrCellRecapUpdateNotFound,
		},
		{
			name: "Error/OtherProjectCell",

			request: request(otherSceneID, otherLayerID, updatedAt),

			expectCell: func(cell *dao.Cell) *dao.Cell { return cell },
			expectErr:  dao.ErrCellRecapUpdateNotFound,
		},
	}

	daoCellRecapUpdate := dao.NewPgCellRecapUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				manuscript, notes := data.cells[0], data.cells[1]

				if testCase.stored != nil {
					data.cells[0] = recapped(manuscript, testCase.stored, testCase.storedAt)
				}

				data.cells = append(data.cells, otherCell)
				insertFixtures(ctx, t, data)

				err := daoCellRecapUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				requireCellsEqual(t, []*dao.Cell{testCase.expectCell(data.cells[0]), notes}, listCells(ctx, t, projectID))
				requireCellsEqual(t, []*dao.Cell{otherCell}, listCells(ctx, t, otherProjectID))
			})
		})
	}
}
