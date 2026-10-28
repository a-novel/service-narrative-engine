package core_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
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

	request := func(ids ...uuid.UUID) *core.SceneDeleteRequest {
		return &core.SceneDeleteRequest{OwnerID: ownerID, ProjectID: projectID, IDs: ids}
	}

	testCases := []struct {
		name string

		request *core.SceneDeleteRequest

		ownerMock  *daoMock[bool]
		lockMock   *daoMock[*dao.Project]
		deleteMock *daoMock[[]*dao.Scene]
		transactor transaction.Transactor

		expect    []*core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: request(sceneID2),

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 1)}},

			expect: []*core.Scene{coreScene(sceneID2, 1)},
		},
		{
			name: "Success/Batch",

			request: request(sceneID3, sceneID1),

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{resp: daoProject()},
			// The delete returns its rows in no particular order.
			deleteMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID3, 2), daoScene(sceneID1, 0)}},

			expect: []*core.Scene{coreScene(sceneID1, 0), coreScene(sceneID3, 2)},
		},
		{
			name: "Error/NoID",

			request: request(),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateID",

			request: request(sceneID2, sceneID2),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyIDs",

			request: request(lo.Times(101, func(int) uuid.UUID { return uuid.New() })...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(sceneID2),

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request(sceneID2),

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Lock",

			request: request(sceneID2),

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(sceneID2, sceneID3),

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{resp: daoProject()},
			// One scene is missing, so the whole delete rolls back.
			deleteMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 1)}},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/Delete",

			request: request(sceneID2),

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[[]*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(sceneID2),

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
					Exec(mock.Anything, &dao.SceneDeleteRequest{ProjectID: projectID, IDs: testCase.request.IDs}).
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
