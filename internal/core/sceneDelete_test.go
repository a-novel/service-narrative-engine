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

	validRequest := &core.SceneDeleteRequest{OwnerID: ownerID, ID: sceneID1}

	testCases := []struct {
		name string

		request    *core.SceneDeleteRequest
		transactor transaction.Transactor

		selectMock   *daoMock[*dao.Scene]
		lockMock     *daoMock[*dao.Project]
		deleteMock   *daoMock[*dao.Scene]
		listMock     *daoMock[[]*dao.Scene]
		reorderOrder []uuid.UUID
		reorderMock  *daoMock[[]*dao.Scene]

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: validRequest,

			selectMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			// The remaining scenes keep their order and close the gap.
			listMock:     &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 1), daoScene(sceneID3, 2)}},
			reorderOrder: []uuid.UUID{sceneID2, sceneID3},
			reorderMock:  &daoMock[[]*dao.Scene]{},

			expect: &core.Scene{ID: sceneID1, ProjectID: projectID, Title: "Scene", Color: sceneColor, UpdatedAt: updatedAt},
		},
		{
			name: "Success/LastScene",

			request: validRequest,

			selectMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			listMock:   &daoMock[[]*dao.Scene]{},

			expect: &core.Scene{ID: sceneID1, ProjectID: projectID, Title: "Scene", Color: sceneColor, UpdatedAt: updatedAt},
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			selectMock: &daoMock[*dao.Scene]{err: dao.ErrSceneSelectNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/List",

			request: validRequest,

			selectMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			listMock:   &daoMock[[]*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Reorder",

			request: validRequest,

			selectMock:   &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			deleteMock:   &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			listMock:     &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 1)}},
			reorderOrder: []uuid.UUID{sceneID2},
			reorderMock:  &daoMock[[]*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request:    validRequest,
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
			mockList := coremocks.NewMockSceneDeleteDaoList(t)
			mockReorder := coremocks.NewMockSceneDeleteDaoReorder(t)

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.SceneSelectRequest{ID: sceneID1, OwnerID: ownerID}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.SceneDeleteRequest{ID: sceneID1, OwnerID: ownerID}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
			}

			if testCase.listMock != nil {
				mockList.EXPECT().
					Exec(mock.Anything, &dao.SceneListRequest{ProjectID: projectID}).
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

			service := core.NewSceneDelete(mockSelect, mockLock, mockDelete, mockList, mockReorder, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockSelect.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
			mockList.AssertExpectations(t)
			mockReorder.AssertExpectations(t)
		})
	}
}
