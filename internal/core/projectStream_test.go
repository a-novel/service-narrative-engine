package core_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// streamCells is how many cells the project stream test reads: one past a full page of 100.
const streamCells = 101

// projectStreamMocks are the reads of a project stream.
type projectStreamMocks struct {
	owner   *coremocks.MockProjectStreamDaoOwner
	project *coremocks.MockProjectStreamDaoProject
	beats   *coremocks.MockProjectStreamDaoBeats
	scenes  *coremocks.MockProjectStreamDaoScenes
	layers  *coremocks.MockProjectStreamDaoLayers
	cells   *coremocks.MockProjectStreamDaoCells
}

// recordName names a record by its position, as a test reads it.
func recordName(record *core.ProjectRecord) string {
	return record.Position.Section + ":" + strconv.Itoa(record.Position.Count)
}

func TestProjectStream(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	cellRows := lo.Times(streamCells, func(index int) *dao.CellListRow {
		return &dao.CellListRow{Cell: *snapshotCell(index), Total: streamCells}
	})
	cellNames := lo.Times(streamCells, func(index int) string { return "cells:" + strconv.Itoa(index+1) })

	expectOwner := func(mocks *projectStreamMocks) {
		mocks.owner.EXPECT().
			Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
			Return(true, nil)
	}
	// expectFrom expects the reads of every section from the scenes on, the scenes from offset.
	expectFrom := func(mocks *projectStreamMocks, offset int) {
		mocks.scenes.EXPECT().
			Exec(mock.Anything, &dao.SceneListRequest{ProjectID: projectID, Limit: 100, Offset: offset}).
			Return([]*dao.SceneListRow{{Scene: *daoScene(sceneID2, 1), Total: 2}}, nil)
		mocks.layers.EXPECT().
			Exec(mock.Anything, &dao.LayerListRequest{ProjectID: projectID, Limit: 100}).
			Return([]*dao.LayerListRow{{Layer: *notesLayer(layerID1, 0), Total: 1}}, nil)
		mocks.cells.EXPECT().
			Exec(mock.Anything, &dao.CellListRequest{ProjectID: projectID, Limit: 100}).
			Return(cellRows[:100], nil)
		mocks.cells.EXPECT().
			Exec(mock.Anything, &dao.CellListRequest{ProjectID: projectID, Limit: 100, Offset: 100}).
			Return(cellRows[100:], nil)
	}

	testCases := []struct {
		name string

		request *core.ProjectStreamRequest
		setup   func(mocks *projectStreamMocks)

		expectRecords []string
		// expectErr is returned before the stream opens.
		expectErr error
		// expectStreamErr cuts the stream after its records.
		expectStreamErr error
	}{
		{
			name: "Success",

			request: &core.ProjectStreamRequest{OwnerID: ownerID, ProjectID: projectID},
			setup: func(mocks *projectStreamMocks) {
				expectOwner(mocks)
				mocks.project.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID}).
					Return(daoProject(), nil)
				mocks.beats.EXPECT().
					Exec(mock.Anything, &dao.BeatListRequest{ProjectID: projectID, Limit: 100}).
					Return([]*dao.BeatListRow{{Beat: *daoBeat(beatID1, 0), Total: 1}}, nil)
				mocks.scenes.EXPECT().
					Exec(mock.Anything, &dao.SceneListRequest{ProjectID: projectID, Limit: 100}).
					Return([]*dao.SceneListRow{{Scene: *daoScene(sceneID1, 0), Total: 1}}, nil)
				mocks.layers.EXPECT().
					Exec(mock.Anything, &dao.LayerListRequest{ProjectID: projectID, Limit: 100}).
					Return([]*dao.LayerListRow{{Layer: *notesLayer(layerID1, 0), Total: 1}}, nil)
				mocks.cells.EXPECT().
					Exec(mock.Anything, &dao.CellListRequest{ProjectID: projectID, Limit: 100}).
					Return(cellRows[:100], nil)
				mocks.cells.EXPECT().
					Exec(mock.Anything, &dao.CellListRequest{ProjectID: projectID, Limit: 100, Offset: 100}).
					Return(cellRows[100:], nil)
			},

			expectRecords: append(
				append([]string{"project:1", "beats:1", "scenes:1", "layers:1"}, cellNames...), "end:0",
			),
		},
		{
			name: "Success/ResumeInSection",

			request: &core.ProjectStreamRequest{
				OwnerID: ownerID, ProjectID: projectID,
				After: &core.ProjectStreamPosition{Section: core.ProjectStreamScenes, Count: 1},
			},
			setup: func(mocks *projectStreamMocks) {
				expectOwner(mocks)
				expectFrom(mocks, 1)
			},

			expectRecords: append(append([]string{"scenes:2", "layers:1"}, cellNames...), "end:0"),
		},
		{
			name: "Success/ResumeAfterSection",

			request: &core.ProjectStreamRequest{
				OwnerID: ownerID, ProjectID: projectID,
				After: &core.ProjectStreamPosition{Section: core.ProjectStreamProject, Count: 1},
			},
			setup: func(mocks *projectStreamMocks) {
				expectOwner(mocks)
				mocks.beats.EXPECT().
					Exec(mock.Anything, &dao.BeatListRequest{ProjectID: projectID, Limit: 100}).
					Return(nil, nil)
				expectFrom(mocks, 0)
			},

			expectRecords: append(append([]string{"scenes:1", "layers:1"}, cellNames...), "end:0"),
		},
		{
			name: "Error/Ended",

			request: &core.ProjectStreamRequest{
				OwnerID: ownerID, ProjectID: projectID, After: &core.ProjectStreamPosition{Section: core.ProjectStreamEnd},
			},
			setup: expectOwner,

			expectErr: core.ErrStreamEnded,
		},
		{
			name: "Error/NotFound",

			request: &core.ProjectStreamRequest{OwnerID: ownerID, ProjectID: projectID},
			setup: func(mocks *projectStreamMocks) {
				mocks.owner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(false, nil)
			},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/UnknownSection",

			request: &core.ProjectStreamRequest{
				OwnerID: ownerID, ProjectID: projectID, After: &core.ProjectStreamPosition{Section: "chapters"},
			},
			setup: func(*projectStreamMocks) {},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ListFails",

			request: &core.ProjectStreamRequest{
				OwnerID: ownerID, ProjectID: projectID,
				After: &core.ProjectStreamPosition{Section: core.ProjectStreamLayers, Count: 1},
			},
			setup: func(mocks *projectStreamMocks) {
				expectOwner(mocks)
				mocks.layers.EXPECT().
					Exec(mock.Anything, &dao.LayerListRequest{ProjectID: projectID, Limit: 100, Offset: 1}).
					Return(nil, errFoo)
			},

			expectStreamErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mocks := &projectStreamMocks{
				owner:   coremocks.NewMockProjectStreamDaoOwner(t),
				project: coremocks.NewMockProjectStreamDaoProject(t),
				beats:   coremocks.NewMockProjectStreamDaoBeats(t),
				scenes:  coremocks.NewMockProjectStreamDaoScenes(t),
				layers:  coremocks.NewMockProjectStreamDaoLayers(t),
				cells:   coremocks.NewMockProjectStreamDaoCells(t),
			}
			testCase.setup(mocks)

			service := core.NewProjectStream(mocks.owner, mocks.project, mocks.beats, mocks.scenes, mocks.layers, mocks.cells)

			records, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			var (
				received  []string
				streamErr error
			)

			if err == nil {
				for record, recordErr := range records {
					if recordErr != nil {
						streamErr = recordErr

						break
					}

					received = append(received, recordName(record))
				}
			}

			require.Equal(t, testCase.expectRecords, received)
			require.ErrorIs(t, streamErr, testCase.expectStreamErr)

			mocks.owner.AssertExpectations(t)
			mocks.project.AssertExpectations(t)
			mocks.beats.AssertExpectations(t)
			mocks.scenes.AssertExpectations(t)
			mocks.layers.AssertExpectations(t)
			mocks.cells.AssertExpectations(t)
		})
	}
}
