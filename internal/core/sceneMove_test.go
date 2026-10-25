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

	existing := []*dao.Scene{daoScene(sceneID1, 0), daoScene(sceneID2, 1), daoScene(sceneID3, 2)}

	coreScene := func(id uuid.UUID, position int) *core.Scene {
		return &core.Scene{
			ID: id, ProjectID: projectID, Position: position, Title: "Scene", Revision: 1, UpdatedAt: updatedAt,
		}
	}

	testCases := []struct {
		name string

		request    *core.SceneMoveRequest
		transactor transaction.Transactor

		selectMock   *daoMock[*dao.Scene]
		lockMock     *daoMock[*dao.Project]
		listMock     *daoMock[[]*dao.Scene]
		reorderOrder []uuid.UUID
		reorderMock  *daoMock[[]*dao.Scene]

		expect    []*core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: &core.SceneMoveRequest{OwnerID: ownerID, ID: sceneID3, After: sceneID1},

			selectMock:   &daoMock[*dao.Scene]{resp: daoScene(sceneID3, 2)},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			listMock:     &daoMock[[]*dao.Scene]{resp: existing},
			reorderOrder: []uuid.UUID{sceneID1, sceneID3, sceneID2},
			// The reorder returns its rows in no particular order.
			reorderMock: &daoMock[[]*dao.Scene]{
				resp: []*dao.Scene{daoScene(sceneID2, 2), daoScene(sceneID1, 0), daoScene(sceneID3, 1)},
			},

			expect: []*core.Scene{coreScene(sceneID1, 0), coreScene(sceneID3, 1), coreScene(sceneID2, 2)},
		},
		{
			name: "Success/First",

			request: &core.SceneMoveRequest{OwnerID: ownerID, ID: sceneID2},

			selectMock:   &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			listMock:     &daoMock[[]*dao.Scene]{resp: existing},
			reorderOrder: []uuid.UUID{sceneID2, sceneID1, sceneID3},
			reorderMock: &daoMock[[]*dao.Scene]{
				resp: []*dao.Scene{daoScene(sceneID2, 0), daoScene(sceneID1, 1), daoScene(sceneID3, 2)},
			},

			expect: []*core.Scene{coreScene(sceneID2, 0), coreScene(sceneID1, 1), coreScene(sceneID3, 2)},
		},
		{
			name: "Error/MissingID",

			request: &core.SceneMoveRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: &core.SceneMoveRequest{OwnerID: ownerID, ID: sceneID3},

			selectMock: &daoMock[*dao.Scene]{err: dao.ErrSceneSelectNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/AfterItself",

			request: &core.SceneMoveRequest{OwnerID: ownerID, ID: sceneID3, After: sceneID3},

			selectMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID3, 2)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			listMock:   &daoMock[[]*dao.Scene]{resp: existing},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Lock",

			request: &core.SceneMoveRequest{OwnerID: ownerID, ID: sceneID3},

			selectMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID3, 2)},
			lockMock:   &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Reorder",

			request: &core.SceneMoveRequest{OwnerID: ownerID, ID: sceneID3, After: sceneID1},

			selectMock:   &daoMock[*dao.Scene]{resp: daoScene(sceneID3, 2)},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			listMock:     &daoMock[[]*dao.Scene]{resp: existing},
			reorderOrder: []uuid.UUID{sceneID1, sceneID3, sceneID2},
			reorderMock:  &daoMock[[]*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request:    &core.SceneMoveRequest{OwnerID: ownerID, ID: sceneID3},
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockSelect := coremocks.NewMockSceneMoveDaoSelect(t)
			mockLock := coremocks.NewMockSceneMoveDaoLock(t)
			mockList := coremocks.NewMockSceneMoveDaoList(t)
			mockReorder := coremocks.NewMockSceneMoveDaoReorder(t)

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.SceneSelectRequest{ID: testCase.request.ID, OwnerID: ownerID}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.listMock != nil {
				mockList.EXPECT().
					Exec(mock.Anything, &dao.SceneListRequest{ProjectID: projectID, Limit: testLimits.MaxScenes}).
					Return(testCase.listMock.resp, testCase.listMock.err)
			}

			if testCase.reorderMock != nil {
				mockReorder.EXPECT().
					Exec(mock.Anything, &dao.SceneReorderRequest{ProjectID: projectID, IDs: testCase.reorderOrder}).
					Return(testCase.reorderMock.resp, testCase.reorderMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewSceneMove(mockSelect, mockLock, mockList, mockReorder, transactor, testLimits)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockSelect.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockList.AssertExpectations(t)
			mockReorder.AssertExpectations(t)
		})
	}
}
