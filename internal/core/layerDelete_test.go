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
		kindMock   *daoMock[string]
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
			kindMock:   &daoMock[string]{resp: core.LayerKindNotes},
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

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[string]{err: dao.ErrLayerKindNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/RequiredLayer",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[string]{resp: core.LayerKindManuscript},

			expectErr: core.ErrRequiredLayer,
		},
		{
			name: "Error/Lock",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[string]{resp: core.LayerKindNotes},
			lockMock:  &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Delete",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			kindMock:   &daoMock[string]{resp: core.LayerKindNotes},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			kindMock:   &daoMock[string]{resp: core.LayerKindNotes},
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockLayerDeleteDaoOwner(t)
			mockKind := coremocks.NewMockLayerDeleteDaoKind(t)
			mockLock := coremocks.NewMockLayerDeleteDaoLock(t)
			mockDelete := coremocks.NewMockLayerDeleteDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.kindMock != nil {
				mockKind.EXPECT().
					Exec(mock.Anything, &dao.LayerKindRequest{ProjectID: projectID, ID: layerID2}).
					Return(testCase.kindMock.resp, testCase.kindMock.err)
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

			service := core.NewLayerDelete(mockOwner, mockKind, mockLock, mockDelete, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockKind.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
		})
	}
}
