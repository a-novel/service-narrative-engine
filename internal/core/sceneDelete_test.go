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

func TestSceneDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.SceneDeleteRequest{OwnerID: ownerID, ProjectID: projectID, ID: sceneID2}

	testCases := []struct {
		name string

		request *core.SceneDeleteRequest

		ownerMock  *daoMock[bool]
		lockMock   *daoMock[*dao.Project]
		deleteMock *daoMock[*dao.Scene]
		transactor transaction.Transactor

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},

			expect: coreScene(sceneID2, 1),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.SceneDeleteRequest{OwnerID: ownerID, ProjectID: projectID},

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
			name: "Error/Lock",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Scene]{err: dao.ErrSceneDeleteNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/Transaction",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockSceneDeleteDaoOwner(t)
			mockLock := coremocks.NewMockSceneDeleteDaoLock(t)
			mockDelete := coremocks.NewMockSceneDeleteDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.SceneDeleteRequest{ProjectID: projectID, ID: sceneID2}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewSceneDelete(mockOwner, mockLock, mockDelete, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
		})
	}
}
