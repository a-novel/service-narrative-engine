package core_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestLayerMove(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	existing := []*dao.Layer{notesLayer(layerID1, 0), notesLayer(layerID2, 1), notesLayer(layerID3, 2)}

	coreLayer := func(id uuid.UUID, position int) *core.Layer {
		return &core.Layer{
			ID: id, ProjectID: projectID, Position: position, Kind: core.LayerKindNotes, Name: "Layer",
			Revision: 1, UpdatedAt: updatedAt,
		}
	}

	testCases := []struct {
		name string

		request    *core.LayerMoveRequest
		transactor transaction.Transactor

		selectMock   *daoMock[*dao.Layer]
		lockMock     *daoMock[*dao.Project]
		listMock     *daoMock[[]*dao.Layer]
		reorderOrder []uuid.UUID
		reorderMock  *daoMock[[]*dao.Layer]

		expect    []*core.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: &core.LayerMoveRequest{OwnerID: ownerID, ID: layerID3, After: layerID1},

			selectMock:   &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			listMock:     &daoMock[[]*dao.Layer]{resp: existing},
			reorderOrder: []uuid.UUID{layerID1, layerID3, layerID2},
			// The reorder returns its rows in no particular order.
			reorderMock: &daoMock[[]*dao.Layer]{
				resp: []*dao.Layer{notesLayer(layerID2, 2), notesLayer(layerID1, 0), notesLayer(layerID3, 1)},
			},

			expect: []*core.Layer{coreLayer(layerID1, 0), coreLayer(layerID3, 1), coreLayer(layerID2, 2)},
		},
		{
			name: "Success/First",

			request: &core.LayerMoveRequest{OwnerID: ownerID, ID: layerID2},

			selectMock:   &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			listMock:     &daoMock[[]*dao.Layer]{resp: existing},
			reorderOrder: []uuid.UUID{layerID2, layerID1, layerID3},
			reorderMock: &daoMock[[]*dao.Layer]{
				resp: []*dao.Layer{notesLayer(layerID2, 0), notesLayer(layerID1, 1), notesLayer(layerID3, 2)},
			},

			expect: []*core.Layer{coreLayer(layerID2, 0), coreLayer(layerID1, 1), coreLayer(layerID3, 2)},
		},
		{
			name: "Error/MissingID",

			request: &core.LayerMoveRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: &core.LayerMoveRequest{OwnerID: ownerID, ID: layerID3},

			selectMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/AfterItself",

			request: &core.LayerMoveRequest{OwnerID: ownerID, ID: layerID3, After: layerID3},

			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			listMock:   &daoMock[[]*dao.Layer]{resp: existing},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Lock",

			request: &core.LayerMoveRequest{OwnerID: ownerID, ID: layerID3},

			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},
			lockMock:   &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Reorder",

			request: &core.LayerMoveRequest{OwnerID: ownerID, ID: layerID3, After: layerID1},

			selectMock:   &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			listMock:     &daoMock[[]*dao.Layer]{resp: existing},
			reorderOrder: []uuid.UUID{layerID1, layerID3, layerID2},
			reorderMock:  &daoMock[[]*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request:    &core.LayerMoveRequest{OwnerID: ownerID, ID: layerID3},
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockSelect := coremocks.NewMockLayerMoveDaoSelect(t)
			mockLock := coremocks.NewMockLayerMoveDaoLock(t)
			mockList := coremocks.NewMockLayerMoveDaoList(t)
			mockReorder := coremocks.NewMockLayerMoveDaoReorder(t)

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ID: testCase.request.ID, OwnerID: ownerID}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

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

			if testCase.reorderMock != nil {
				mockReorder.EXPECT().
					Exec(mock.Anything, &dao.LayerReorderRequest{ProjectID: projectID, IDs: testCase.reorderOrder}).
					Return(testCase.reorderMock.resp, testCase.reorderMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerMove(mockSelect, mockLock, mockList, mockReorder, transactor, testLimits)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockSelect.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockList.AssertExpectations(t)
			mockReorder.AssertExpectations(t)
		})
	}
}
