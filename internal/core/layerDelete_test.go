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

func TestLayerDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.LayerDeleteRequest{OwnerID: ownerID, ID: layerID2}

	testCases := []struct {
		name string

		request *core.LayerDeleteRequest

		selectMock *daoMock[*dao.Layer]
		lockMock   *daoMock[*dao.Project]
		deleteMock *daoMock[*dao.Layer]
		transactor transaction.Transactor

		expect    *core.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},

			expect: coreLayer(layerID2, 1),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.LayerDeleteRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: request,

			selectMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/ManuscriptLayer",

			request: &core.LayerDeleteRequest{OwnerID: ownerID, ID: layerID2},

			selectMock: &daoMock[*dao.Layer]{resp: daoLayer(layerID2, 1, core.LayerKindManuscript)},

			expectErr: core.ErrManuscriptLayer,
		},
		{
			name: "Error/OtherOwner",

			request: request,

			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: foreignProject()},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/DeletedMeanwhile",

			request: request,

			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Layer]{err: dao.ErrLayerDeleteNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Transaction",

			request: request,

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

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ID: layerID2}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.LayerDeleteRequest{ID: layerID2}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerDelete(mockSelect, mockLock, mockDelete, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockSelect.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
		})
	}
}
