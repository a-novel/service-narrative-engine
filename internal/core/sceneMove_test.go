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

func TestSceneMove(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := func(position int) *core.SceneMoveRequest {
		return &core.SceneMoveRequest{
			OwnerID: ownerID, ProjectID: projectID, ID: sceneID2, BeatID: beatID2, Position: position,
		}
	}

	testCases := []struct {
		name string

		request *core.SceneMoveRequest

		lockMock   *daoMock[bool]
		moveMock   *daoMock[*dao.Scene]
		transactor transaction.Transactor

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success/Down",

			request: request(2),

			lockMock: &daoMock[bool]{resp: true},
			moveMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 2)},

			expect: coreScene(sceneID2, 2),
		},
		{
			name: "Success/First",

			request: request(0),

			lockMock: &daoMock[bool]{resp: true},
			moveMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 0)},

			expect: coreScene(sceneID2, 0),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.SceneMoveRequest{OwnerID: ownerID, ProjectID: projectID, ID: sceneID2},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativePosition",

			request: request(-1),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(0),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(0),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(0),

			lockMock: &daoMock[bool]{resp: true},
			moveMock: &daoMock[*dao.Scene]{err: dao.ErrSceneMoveNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/BeatNotFound",

			request: request(0),

			lockMock: &daoMock[bool]{resp: true},
			moveMock: &daoMock[*dao.Scene]{err: dao.ErrSceneMoveBeatNotFound},

			expectErr: core.ErrBeatNotFound,
		},
		{
			name: "Error/BeatFull",

			request: request(0),

			lockMock: &daoMock[bool]{resp: true},
			moveMock: &daoMock[*dao.Scene]{err: dao.ErrSceneMoveFull},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/Move",

			request: request(0),

			lockMock: &daoMock[bool]{resp: true},
			moveMock: &daoMock[*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(0),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockSceneMoveDaoLock(t)
			mockMove := coremocks.NewMockSceneMoveDao(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.moveMock != nil {
				mockMove.EXPECT().
					Exec(mock.Anything, &dao.SceneMoveRequest{
						ProjectID: projectID, ID: sceneID2, BeatID: beatID2, Position: testCase.request.Position,
					}).
					Return(testCase.moveMock.resp, testCase.moveMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewSceneMove(mockLock, mockMove, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockMove.AssertExpectations(t)
		})
	}
}
