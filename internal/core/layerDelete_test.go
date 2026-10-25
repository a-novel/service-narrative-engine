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

func TestLayerDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := &core.LayerDeleteRequest{OwnerID: ownerID, ID: layerID1, Revision: 1}

	testCases := []struct {
		name string

		request    *core.LayerDeleteRequest
		transactor transaction.Transactor

		selectMock   *daoMock[*dao.Layer]
		lockMock     *daoMock[*dao.Project]
		deleteMock   *daoMock[*dao.Layer]
		listMock     *daoMock[[]*dao.Layer]
		reorderOrder []uuid.UUID
		reorderMock  *daoMock[[]*dao.Layer]

		expect    *core.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: validRequest,

			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID1, 0)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID1, 0)},
			// The remaining layers keep their order and close the gap.
			listMock:     &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID2, 1), notesLayer(layerID3, 2)}},
			reorderOrder: []uuid.UUID{layerID2, layerID3},
			reorderMock:  &daoMock[[]*dao.Layer]{},

			expect: &core.Layer{
				ID: layerID1, ProjectID: projectID, Kind: core.LayerKindNotes, Name: "Layer", Revision: 1,
				UpdatedAt: updatedAt,
			},
		},
		{
			name: "Success/LastLayer",

			request: validRequest,

			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID1, 0)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID1, 0)},
			listMock:   &daoMock[[]*dao.Layer]{},

			expect: &core.Layer{
				ID: layerID1, ProjectID: projectID, Kind: core.LayerKindNotes, Name: "Layer", Revision: 1,
				UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/MissingRevision",

			request: &core.LayerDeleteRequest{OwnerID: ownerID, ID: layerID1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ManuscriptLayer",

			request: validRequest,

			selectMock: &daoMock[*dao.Layer]{resp: daoLayer(layerID1, 0, core.LayerKindManuscript)},

			expectErr: core.ErrManuscriptLayer,
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			selectMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/RevisionConflict",

			request: validRequest,

			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID1, 0)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Layer]{err: dao.ErrLayerDeleteNotFound},

			expectErr: core.ErrRevisionConflict,
		},
		{
			name: "Error/List",

			request: validRequest,

			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID1, 0)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID1, 0)},
			listMock:   &daoMock[[]*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Reorder",

			request: validRequest,

			selectMock:   &daoMock[*dao.Layer]{resp: notesLayer(layerID1, 0)},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock:   &daoMock[*dao.Layer]{resp: notesLayer(layerID1, 0)},
			listMock:     &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID2, 1)}},
			reorderOrder: []uuid.UUID{layerID2},
			reorderMock:  &daoMock[[]*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request:    validRequest,
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockSelect := coremocks.NewMockLayerDeleteDaoSelect(t)
			mockLock := coremocks.NewMockLayerDeleteDaoLock(t)
			mockDelete := coremocks.NewMockLayerDeleteDao(t)
			mockList := coremocks.NewMockLayerDeleteDaoList(t)
			mockReorder := coremocks.NewMockLayerDeleteDaoReorder(t)

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ID: layerID1, OwnerID: ownerID}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.LayerDeleteRequest{ID: layerID1, OwnerID: ownerID, Revision: 1}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
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

			service := core.NewLayerDelete(mockSelect, mockLock, mockDelete, mockList, mockReorder, transactor, testLimits)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockSelect.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
			mockList.AssertExpectations(t)
			mockReorder.AssertExpectations(t)
		})
	}
}
