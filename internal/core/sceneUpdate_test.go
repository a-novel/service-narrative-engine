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

func TestSceneUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.SceneUpdateRequest{
		OwnerID: ownerID, ProjectID: projectID, ID: sceneID2, Title: "Scene",
	}

	testCases := []struct {
		name string

		request *core.SceneUpdateRequest

		lockMock   *daoMock[bool]
		updateMock *daoMock[*dao.Scene]
		bytesMock  *daoMock[*dao.ProjectBytes]
		transactor transaction.Transactor

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			lockMock:   &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			bytesMock:  withinBudget,

			expect: coreScene(sceneID2, 1),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.SceneUpdateRequest{OwnerID: ownerID, ProjectID: projectID, ID: sceneID2},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request,

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request,

			lockMock:   &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Scene]{err: dao.ErrSceneUpdateNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/Update",

			request: request,

			lockMock:   &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: request,

			lockMock:   &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			bytesMock:  overBudget,

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/ProjectBytes",

			request: request,

			lockMock:   &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			bytesMock:  &daoMock[*dao.ProjectBytes]{err: errFoo},

			expectErr: errFoo,
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

			mockLock := coremocks.NewMockSceneUpdateDaoLock(t)
			mockUpdate := coremocks.NewMockSceneUpdateDao(t)
			mockBytes := coremocks.NewMockSceneUpdateDaoBytes(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.SceneUpdateRequest) bool {
						return req.ProjectID == projectID && req.ID == sceneID2 &&
							req.Title == "Scene" && recent(req.Now)
					})).
					Return(testCase.updateMock.resp, testCase.updateMock.err)
			}

			if testCase.bytesMock != nil {
				mockBytes.EXPECT().
					Exec(mock.Anything, &dao.ProjectBytesRequest{ID: projectID}).
					Return(testCase.bytesMock.resp, testCase.bytesMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewSceneUpdate(mockLock, mockUpdate, mockBytes, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
			mockBytes.AssertExpectations(t)
		})
	}
}
