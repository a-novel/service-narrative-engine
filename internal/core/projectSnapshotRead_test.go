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

	cellRows := lo.Times(snapshotCells, func(index int) *dao.CellSummaryListRow {
		return &dao.CellSummaryListRow{
			SceneID: sceneID1, LayerID: snapshotCell(index).LayerID, UpdatedAt: updatedAt, Size: 6, Speakers: []string{},
			Total: snapshotCells,
		}
	})

	// The first cell holds a recap, and the beat too, which the snapshot also lists apart for the planner.
	cellRows[0].RecapNames, cellRows[0].RecapCellUpdatedAt = []string{"Marie"}, &createdAt
	cellRows[0].RecapCondensedSize, cellRows[0].RecapCompactSize, cellRows[0].RecapEssentialsSize = lo.ToPtr(9),
		lo.ToPtr(7), lo.ToPtr(5)

	recap := &dao.Recap{Condensed: "Notes.", Compact: "Notes.", Essentials: "Notes.", Names: []string{"Marie"}}

	recappedBeat := daoBeat(beatID1, 0)
	recappedBeat.Recap, recappedBeat.RecapBeatUpdatedAt = recap, &createdAt
	recappedBeat.RecapSources = []*dao.BeatRecapSource{{SceneID: sceneID1, CellUpdatedAt: createdAt}, {SceneID: sceneID2}}

	coreRecap := core.Recap{Condensed: "Notes.", Compact: "Notes.", Essentials: "Notes.", Names: []string{"Marie"}}
	sources := []*core.BeatRecapSource{{SceneID: sceneID1, CellUpdatedAt: createdAt}, {SceneID: sceneID2}}

	snapshot := &core.ProjectSnapshot{
		Project: &core.Project{
			ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr", Tags: []string{"noir"},
			Genres:    []string{"thriller"},
			CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt,
			LogoChecksum: lo.ToPtr(logoChecksum),
		},
		Beats: []*core.Beat{with(coreBeat(beatID1, 0), func(beat *core.Beat) {
			beat.Recap, beat.RecapBeatUpdatedAt, beat.RecapSources = &coreRecap, createdAt, sources
		})},
		Scenes: []*core.Scene{coreScene(sceneID1, 0), coreScene(sceneID2, 1)},
		Layers: []*core.Layer{coreLayer(layerID1, 0)},
		Cells: lo.Times(snapshotCells, func(index int) *core.CellSummary {
			summary := &core.CellSummary{
				SceneID: sceneID1, LayerID: snapshotCell(index).LayerID, UpdatedAt: updatedAt, Size: 6, Speakers: []string{},
			}
			if index == 0 {
				summary.Recap = &core.RecapSummary{
					Names: []string{"Marie"}, CondensedSize: 9, CompactSize: 7, EssentialsSize: 5, CellUpdatedAt: createdAt,
				}
			}

			return summary
		}),
		BeatRecaps: []*core.BeatRecap{{
			Recap: coreRecap, BeatID: beatID1, BeatUpdatedAt: createdAt, Sources: sources,
		}},
	}

	testCases := []struct {
		name string

		request *core.ProjectSnapshotReadRequest

		ownerMock *daoMock[bool]
		bytesMock *daoMock[*dao.ProjectBytes]
		// read reads the project, its beats, scenes and layers, and cellsErr fails the cell list.
		read     bool
		cellsErr error
		// cellPages are the sizes of the pages of cells the list returns, which hold every cell together.
		cellPages []int

		expect    *core.ProjectSnapshot
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ProjectSnapshotReadRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: true},
			bytesMock: withinBudget,
			read:      true,
			// A full page of cells asks for the next one.
			cellPages: []int{2000, 1},

			expect: snapshot,
		},
		{
			// A page that stops short of its limit, as a byte bound stops a page, still asks for the next one,
			// until the pages hold the total.
			name: "Success/ShortPages",

			request: &core.ProjectSnapshotReadRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: true},
			bytesMock: withinBudget,
			read:      true,
			cellPages: []int{1500, 1, 500},

			expect: snapshot,
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
			// A project past its budget would take a context past every call's limit, and no page is read.
			name: "Error/ProjectBudget",

			request: &core.ProjectSnapshotReadRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: true},
			bytesMock: overBudget,

			expectErr: core.ErrContextTooLarge,
		},
		{
			name: "Error/ProjectBytes",

			request: &core.ProjectSnapshotReadRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: true},
			bytesMock: &daoMock[*dao.ProjectBytes]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Cells",

			request: &core.ProjectSnapshotReadRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: true},
			bytesMock: withinBudget,
			read:      true,
			cellsErr:  errFoo,

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockProjectSnapshotReadDaoOwner(t)
			mockBytes := coremocks.NewMockProjectSnapshotReadDaoBytes(t)
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

			if testCase.bytesMock != nil {
				mockBytes.EXPECT().
					Exec(mock.Anything, &dao.ProjectBytesRequest{ID: projectID}).
					Return(testCase.bytesMock.resp, testCase.bytesMock.err)
			}

			if testCase.read {
				mockProject.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID}).
					Return(daoProject(), nil)
				mockBeats.EXPECT().
					Exec(mock.Anything, &dao.BeatListRequest{ProjectID: projectID, Limit: 2000, MaxBytes: core.ListPageBytesMax}).
					Return([]*dao.BeatListRow{{Beat: *recappedBeat, Total: 1}}, nil)
				mockScenes.EXPECT().
					Exec(mock.Anything, &dao.SceneListRequest{ProjectID: projectID, Limit: 2000, MaxBytes: core.ListPageBytesMax}).
					Return([]*dao.SceneListRow{
						{Scene: *daoScene(sceneID1, 0), Total: 2}, {Scene: *daoScene(sceneID2, 1), Total: 2},
					}, nil)
				mockLayers.EXPECT().
					Exec(mock.Anything, &dao.LayerListRequest{ProjectID: projectID, Limit: 2000, MaxBytes: core.ListPageBytesMax}).
					Return([]*dao.LayerListRow{{Layer: *notesLayer(layerID1, 0), Total: 1}}, nil)
			}

			var offset int

			for _, size := range testCase.cellPages {
				mockCells.EXPECT().
					Exec(mock.Anything, &dao.CellSummaryListRequest{ProjectID: projectID, Limit: 2000, Offset: offset}).
					Return(cellRows[offset:offset+size], nil).
					Once()

				offset += size
			}

			if testCase.cellsErr != nil {
				mockCells.EXPECT().
					Exec(mock.Anything, &dao.CellSummaryListRequest{ProjectID: projectID, Limit: 2000}).
					Return(nil, testCase.cellsErr)
			}

			service := core.NewProjectSnapshotRead(
				mockOwner, mockBytes, mockProject, mockBeats, mockScenes, mockLayers, mockCells,
			)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockBytes.AssertExpectations(t)
			mockProject.AssertExpectations(t)
			mockBeats.AssertExpectations(t)
			mockScenes.AssertExpectations(t)
			mockLayers.AssertExpectations(t)
			mockCells.AssertExpectations(t)
		})
	}
}
