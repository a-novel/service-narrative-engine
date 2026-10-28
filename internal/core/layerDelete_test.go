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

	request := &core.LayerDeleteRequest{OwnerID: ownerID, ProjectID: projectID, ID: layerID2}

	testCases := []struct {
		name string

		request *core.LayerDeleteRequest

		ownerMock  *daoMock[bool]
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

			ownerMock:  &daoMock[bool]{resp: true},
			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},

			expect: coreLayer(layerID2, 1),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.LayerDeleteRequest{OwnerID: ownerID, ProjectID: projectID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request,

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			selectMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/ManuscriptLayer",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			selectMock: &daoMock[*dao.Layer]{resp: daoLayer(layerID2, 1, core.LayerKindManuscript)},

			expectErr: core.ErrManuscriptLayer,
		},
		{
			name: "Error/ProjectNotFound",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{err: dao.ErrProjectLockNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/DeletedMeanwhile",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Layer]{err: dao.ErrLayerDeleteNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Transaction",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockLayerDeleteDaoOwner(t)
			mockSelect := coremocks.NewMockLayerDeleteDaoSelect(t)
			mockLock := coremocks.NewMockLayerDeleteDaoLock(t)
			mockDelete := coremocks.NewMockLayerDeleteDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ProjectID: projectID, ID: layerID2}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.LayerDeleteRequest{ProjectID: projectID, ID: layerID2}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerDelete(mockOwner, mockSelect, mockLock, mockDelete, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockSelect.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
		})
	}
}
