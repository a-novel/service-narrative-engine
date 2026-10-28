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

	request := &core.SceneDeleteRequest{OwnerID: ownerID, ID: sceneID2}

	testCases := []struct {
		name string

		request *core.SceneDeleteRequest

		selectMock *daoMock[*dao.Scene]
		lockMock   *daoMock[*dao.Project]
		deleteMock *daoMock[*dao.Scene]
		transactor transaction.Transactor

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			selectMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},

			expect: coreScene(sceneID2, 1),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.SceneDeleteRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: request,

			selectMock: &daoMock[*dao.Scene]{err: dao.ErrSceneSelectNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: request,

			selectMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: foreignProject()},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/DeletedMeanwhile",

			request: request,

			selectMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Scene]{err: dao.ErrSceneDeleteNotFound},

			expectErr: core.ErrSceneNotFound,
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

			mockSelect := coremocks.NewMockSceneDeleteDaoSelect(t)
			mockLock := coremocks.NewMockSceneDeleteDaoLock(t)
			mockDelete := coremocks.NewMockSceneDeleteDao(t)

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.SceneSelectRequest{ID: sceneID2}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.SceneDeleteRequest{ID: sceneID2}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewSceneDelete(mockSelect, mockLock, mockDelete, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockSelect.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
		})
	}
}
