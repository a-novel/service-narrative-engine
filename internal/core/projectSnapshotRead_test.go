package core_test

import (
	"encoding/json"
	"errors"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// snapshotCells is how many cells the snapshot test reads: one past a full page of 2,000.
const snapshotCells = 2001

// snapshotCell returns the cell of the first test scene in a layer the index names.
func snapshotCell(index int) *dao.Cell {
	return &dao.Cell{
		ProjectID: projectID, SceneID: sceneID1, LayerID: uuid.NewSHA1(uuid.Nil, []byte(strconv.Itoa(index))),
		Content: json.RawMessage(`{"text":"Notes."}`), UpdatedAt: updatedAt,
	}
}

func TestProjectSnapshotRead(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	cellRows := lo.Times(snapshotCells, func(index int) *dao.CellListRow {
		return &dao.CellListRow{Cell: *snapshotCell(index), Total: snapshotCells}
	})

	// The first cell and the beat hold recaps, which the snapshot also lists apart for the planner.
	recap := &dao.Recap{Condensed: "Notes.", Compact: "Notes.", Essentials: "Notes.", Names: []string{"Marie"}}
	cellRows[0].Recap, cellRows[0].RecapCellUpdatedAt = recap, &createdAt

	recappedBeat := daoBeat(beatID1, 0)
	recappedBeat.Recap, recappedBeat.RecapBeatUpdatedAt = recap, &createdAt
	recappedBeat.RecapSources = []*dao.BeatRecapSource{{SceneID: sceneID1, CellUpdatedAt: createdAt}, {SceneID: sceneID2}}

	coreRecap := core.Recap{Condensed: "Notes.", Compact: "Notes.", Essentials: "Notes.", Names: []string{"Marie"}}
	sources := []*core.BeatRecapSource{{SceneID: sceneID1, CellUpdatedAt: createdAt}, {SceneID: sceneID2}}

	testCases := []struct {
		name string

		request *core.ProjectSnapshotReadRequest

		ownerMock *daoMock[bool]
		// read reads the project, its beats, scenes and layers, and cellsErr fails the cell list.
		read     bool
		cellsErr error

		expect    *core.ProjectSnapshot
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ProjectSnapshotReadRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: true},
			read:      true,

			expect: &core.ProjectSnapshot{
				Project: &core.Project{
					ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
					CreatedAt: createdAt, UpdatedAt: updatedAt,
				},
				Beats: []*core.Beat{with(coreBeat(beatID1, 0), func(beat *core.Beat) {
					beat.Recap, beat.RecapBeatUpdatedAt, beat.RecapSources = &coreRecap, createdAt, sources
				})},
				Scenes: []*core.Scene{coreScene(sceneID1, 0), coreScene(sceneID2, 1)},
				Layers: []*core.Layer{coreLayer(layerID1, 0)},
				Cells: lo.Times(snapshotCells, func(index int) *core.Cell {
					cell := snapshotCell(index)

					loaded := &core.Cell{
						ProjectID: projectID, SceneID: sceneID1, LayerID: cell.LayerID, Content: cell.Content,
						UpdatedAt: updatedAt,
					}
					if index == 0 {
						loaded.Recap, loaded.RecapCellUpdatedAt = &coreRecap, createdAt
					}

					return loaded
				}),
				Recaps: []*core.CellRecap{{
					Recap: coreRecap, SceneID: sceneID1, LayerID: snapshotCell(0).LayerID, CellUpdatedAt: createdAt,
				}},
				BeatRecaps: []*core.BeatRecap{{
					Recap: coreRecap, BeatID: beatID1, BeatUpdatedAt: createdAt, Sources: sources,
				}},
			},
		},
		{
			name: "Error/MissingProject",

			request: &core.ProjectSnapshotReadRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: &core.ProjectSnapshotReadRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Cells",

			request: &core.ProjectSnapshotReadRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: true},
			read:      true,
			cellsErr:  errFoo,

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockProjectSnapshotReadDaoOwner(t)
			mockProject := coremocks.NewMockProjectSnapshotReadDaoProject(t)
			mockBeats := coremocks.NewMockProjectSnapshotReadDaoBeats(t)
			mockScenes := coremocks.NewMockProjectSnapshotReadDaoScenes(t)
			mockLayers := coremocks.NewMockProjectSnapshotReadDaoLayers(t)
			mockCells := coremocks.NewMockProjectSnapshotReadDaoCells(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.read {
				mockProject.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID}).
					Return(daoProject(), nil)
				mockBeats.EXPECT().
					Exec(mock.Anything, &dao.BeatListRequest{ProjectID: projectID, Limit: 2000}).
					Return([]*dao.BeatListRow{{Beat: *recappedBeat, Total: 1}}, nil)
				mockScenes.EXPECT().
					Exec(mock.Anything, &dao.SceneListRequest{ProjectID: projectID, Limit: 2000}).
					Return([]*dao.SceneListRow{
						{Scene: *daoScene(sceneID1, 0), Total: 2}, {Scene: *daoScene(sceneID2, 1), Total: 2},
					}, nil)
				mockLayers.EXPECT().
					Exec(mock.Anything, &dao.LayerListRequest{ProjectID: projectID, Limit: 2000}).
					Return([]*dao.LayerListRow{{Layer: *notesLayer(layerID1, 0), Total: 1}}, nil)
			}

			// A full page of cells asks for the next one.
			if testCase.read && testCase.cellsErr == nil {
				mockCells.EXPECT().
					Exec(mock.Anything, &dao.CellListRequest{ProjectID: projectID, Limit: 2000}).
					Return(cellRows[:2000], nil)
				mockCells.EXPECT().
					Exec(mock.Anything, &dao.CellListRequest{ProjectID: projectID, Limit: 2000, Offset: 2000}).
					Return(cellRows[2000:], nil)
			}

			if testCase.cellsErr != nil {
				mockCells.EXPECT().
					Exec(mock.Anything, &dao.CellListRequest{ProjectID: projectID, Limit: 2000}).
					Return(nil, testCase.cellsErr)
			}

			service := core.NewProjectSnapshotRead(mockOwner, mockProject, mockBeats, mockScenes, mockLayers, mockCells)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockProject.AssertExpectations(t)
			mockBeats.AssertExpectations(t)
			mockScenes.AssertExpectations(t)
			mockLayers.AssertExpectations(t)
			mockCells.AssertExpectations(t)
		})
	}
}
