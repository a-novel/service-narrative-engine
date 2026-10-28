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

func TestSceneDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := func(id uuid.UUID) *core.SceneDeleteRequest {
		return &core.SceneDeleteRequest{OwnerID: ownerID, ProjectID: projectID, ID: id}
	}

	testCases := []struct {
		name string

		request *core.SceneDeleteRequest

		lockMock   *daoMock[bool]
		deleteMock *daoMock[*dao.Scene]
		transactor transaction.Transactor

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: request(sceneID2),

			lockMock:   &daoMock[bool]{resp: true},
			deleteMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},

			expect: coreScene(sceneID2, 1),
		},
		{
			name: "Error/InvalidRequest",

			request: request(uuid.Nil),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(sceneID2),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(sceneID2),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(sceneID2),

			lockMock:   &daoMock[bool]{resp: true},
			deleteMock: &daoMock[*dao.Scene]{err: dao.ErrSceneDeleteNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/Delete",

			request: request(sceneID2),

			lockMock:   &daoMock[bool]{resp: true},
			deleteMock: &daoMock[*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(sceneID2),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockSceneDeleteDaoLock(t)
			mockDelete := coremocks.NewMockSceneDeleteDao(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.SceneDeleteRequest{ProjectID: projectID, ID: testCase.request.ID}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewSceneDelete(mockLock, mockDelete, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
		})
	}
}
