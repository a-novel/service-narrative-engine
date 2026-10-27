package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestLayerCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	existing := []*dao.Layer{notesLayer(layerID1, 0), notesLayer(layerID2, 1)}

	// The new layer's ID is generated, so a reorder is matched by where the existing layers land.
	newBetween := func(req *dao.LayerReorderRequest) bool {
		return len(req.IDs) == 3 && req.IDs[0] == layerID1 && req.IDs[2] == layerID2
	}
	newFirst := func(req *dao.LayerReorderRequest) bool {
		return len(req.IDs) == 3 && req.IDs[1] == layerID1 && req.IDs[2] == layerID2
	}

	testCases := []struct {
		name string

		request    *core.LayerCreateRequest
		transactor transaction.Transactor

		lockMock    *daoMock[*dao.Project]
		listMock    *daoMock[[]*dao.Layer]
		insertMock  *daoMock[*dao.Layer]
		reorderMock *daoMock[[]*dao.Layer]
		reorderedBy func(req *dao.LayerReorderRequest) bool

		expect    *core.Layer
		expectErr error
	}{
		{
			name: "Success/Last",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, After: layerID2, Name: "Layer"},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			listMock:   &daoMock[[]*dao.Layer]{resp: existing},
			insertMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},

			expect: &core.Layer{
				ID: layerID3, ProjectID: projectID, Position: 2, Kind: core.LayerKindNotes, Name: "Layer",
				UpdatedAt: updatedAt,
			},
		},
		{
			name: "Success/Between",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, After: layerID1, Name: "Layer"},

			lockMock:    &daoMock[*dao.Project]{resp: daoProject()},
			listMock:    &daoMock[[]*dao.Layer]{resp: existing},
			insertMock:  &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},
			reorderMock: &daoMock[[]*dao.Layer]{},
			reorderedBy: newBetween,

			expect: &core.Layer{
				ID: layerID3, ProjectID: projectID, Position: 1, Kind: core.LayerKindNotes, Name: "Layer",
				UpdatedAt: updatedAt,
			},
		},
		{
			name: "Success/First",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			lockMock:    &daoMock[*dao.Project]{resp: daoProject()},
			listMock:    &daoMock[[]*dao.Layer]{resp: existing},
			insertMock:  &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},
			reorderMock: &daoMock[[]*dao.Layer]{},
			reorderedBy: newFirst,

			expect: &core.Layer{
				ID: layerID3, ProjectID: projectID, Position: 0, Kind: core.LayerKindNotes, Name: "Layer",
				UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/MissingProject",

			request: &core.LayerCreateRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			lockMock: &daoMock[*dao.Project]{err: dao.ErrProjectLockNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/ProjectFull",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			lockMock: &daoMock[*dao.Project]{resp: daoProject()},
			listMock: &daoMock[[]*dao.Layer]{resp: append(existing, notesLayer(layerID3, 2))},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/UnknownAnchor",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, After: layerID3, Name: "Layer"},

			lockMock: &daoMock[*dao.Project]{resp: daoProject()},
			listMock: &daoMock[[]*dao.Layer]{resp: existing},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Insert",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, After: layerID2, Name: "Layer"},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			listMock:   &daoMock[[]*dao.Layer]{resp: existing},
			insertMock: &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Reorder",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			lockMock:    &daoMock[*dao.Project]{resp: daoProject()},
			listMock:    &daoMock[[]*dao.Layer]{resp: existing},
			insertMock:  &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},
			reorderMock: &daoMock[[]*dao.Layer]{err: errFoo},
			reorderedBy: newFirst,

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request:    &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockLayerCreateDaoLock(t)
			mockList := coremocks.NewMockLayerCreateDaoList(t)
			mockInsert := coremocks.NewMockLayerCreateDaoInsert(t)
			mockReorder := coremocks.NewMockLayerCreateDaoReorder(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.listMock != nil {
				mockList.EXPECT().
					Exec(mock.Anything, &dao.LayerListRequest{ProjectID: projectID, Limit: testLimits.MaxLayers}).
					Return(testCase.listMock.resp, testCase.listMock.err)
			}

			if testCase.insertMock != nil {
				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayerInsertRequest) bool {
						return req.ProjectID == projectID && req.Position == len(existing) &&
							req.Kind == core.LayerKindNotes && req.Name == testCase.request.Name && recent(req.Now)
					})).
					Return(testCase.insertMock.resp, testCase.insertMock.err)
			}

			if testCase.reorderMock != nil {
				mockReorder.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(testCase.reorderedBy)).
					Return(testCase.reorderMock.resp, testCase.reorderMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerCreate(mockLock, mockList, mockInsert, mockReorder, transactor, testLimits)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockList.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
			mockReorder.AssertExpectations(t)
		})
	}
}
