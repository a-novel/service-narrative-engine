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

func TestSceneMove(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := func(after uuid.UUID) *core.SceneMoveRequest {
		return &core.SceneMoveRequest{OwnerID: ownerID, ID: sceneID2, After: after}
	}

	testCases := []struct {
		name string

		request *core.SceneMoveRequest

		// targetMock answers both reads of the moved scene, before and after the lock.
		targetMock *daoMock[*dao.Scene]
		lockMock   *daoMock[*dao.Project]
		anchorMock *daoMock[*dao.Scene]
		// to is the position the service moves the scene to, when the case reaches the move.
		to         int
		moveMock   *daoMock[*dao.Scene]
		transactor transaction.Transactor

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success/Down",

			request: request(sceneID3),

			targetMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID3, 2)},
			to:         2,
			moveMock:   &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 2)},

			expect: coreScene(sceneID2, 2),
		},
		{
			name: "Success/First",

			request: request(uuid.Nil),

			targetMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			to:         0,
			moveMock:   &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 0)},

			expect: coreScene(sceneID2, 0),
		},
		{
			name: "Success/InPlace",

			request: request(sceneID1),

			targetMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},

			expect: coreScene(sceneID2, 1),
		},
		{
			name: "Error/Self",

			request: request(sceneID2),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest",

			request: &core.SceneMoveRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: request(uuid.Nil),

			targetMock: &daoMock[*dao.Scene]{err: dao.ErrSceneSelectNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: request(uuid.Nil),

			targetMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: foreignProject()},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/UnknownAnchor",

			request: request(sceneID3),

			targetMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Scene]{err: dao.ErrSceneSelectNotFound},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/AnchorElsewhere",

			request: request(sceneID3),

			targetMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Scene]{resp: func() *dao.Scene {
				anchor := daoScene(sceneID3, 2)
				anchor.ProjectID = otherProjectID

				return anchor
			}()},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Move",

			request: request(uuid.Nil),

			targetMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			to:         0,
			moveMock:   &daoMock[*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(uuid.Nil),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockSelect := coremocks.NewMockSceneMoveDaoSelect(t)
			mockLock := coremocks.NewMockSceneMoveDaoLock(t)
			mockMove := coremocks.NewMockSceneMoveDao(t)

			if testCase.targetMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.SceneSelectRequest{ID: sceneID2}).
					Return(testCase.targetMock.resp, testCase.targetMock.err)
			}

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.anchorMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.SceneSelectRequest{ID: testCase.request.After}).
					Return(testCase.anchorMock.resp, testCase.anchorMock.err)
			}

			if testCase.moveMock != nil {
				mockMove.EXPECT().
					Exec(mock.Anything, &dao.SceneMoveRequest{ID: sceneID2, To: testCase.to}).
					Return(testCase.moveMock.resp, testCase.moveMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewSceneMove(mockSelect, mockLock, mockMove, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockSelect.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockMove.AssertExpectations(t)
		})
	}
}
