package core_test

import (
	"errors"
	"slices"
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

func TestScenesUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	scene := func(id uuid.UUID) core.ScenesUpdateItem {
		return core.ScenesUpdateItem{ID: id, Title: "Scene"}
	}

	// updateScene is scene as the update receives it.
	updateScene := func(id uuid.UUID) dao.ScenesUpdateItem {
		return dao.ScenesUpdateItem{ID: id, Title: "Scene"}
	}

	request := func(scenes ...core.ScenesUpdateItem) *core.ScenesUpdateRequest {
		return &core.ScenesUpdateRequest{OwnerID: ownerID, ProjectID: projectID, Scenes: scenes}
	}

	testCases := []struct {
		name string

		request *core.ScenesUpdateRequest

		lockMock   *daoMock[bool]
		updateMock *daoMock[[]*dao.Scene]
		bytesMock  *daoMock[*dao.ProjectBytes]
		transactor transaction.Transactor
		// updateScenes are the scenes the update receives.
		updateScenes []dao.ScenesUpdateItem

		expect    []*core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: request(scene(sceneID2)),

			lockMock:     &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 1)}},
			bytesMock:    withinBudget,
			updateScenes: []dao.ScenesUpdateItem{updateScene(sceneID2)},

			expect: []*core.Scene{coreScene(sceneID2, 1)},
		},
		{
			name: "Success/Batch",

			request: request(scene(sceneID3), core.ScenesUpdateItem{
				ID: sceneID1, Title: "Le thème", Description: "Paul ment.", Prompt: "Lent.",
			}),

			lockMock: &daoMock[bool]{resp: true},
			// The update returns its rows in no particular order.
			updateMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID1, 0), daoScene(sceneID3, 2)}},
			bytesMock:  withinBudget,
			updateScenes: []dao.ScenesUpdateItem{
				updateScene(sceneID3),
				{ID: sceneID1, Title: "Le thème", Description: "Paul ment.", Prompt: "Lent."},
			},

			expect: []*core.Scene{coreScene(sceneID3, 2), coreScene(sceneID1, 0)},
		},
		{
			name: "Error/InvalidRequest",

			request: request(core.ScenesUpdateItem{ID: sceneID2, Description: "Paul ment."}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoScene",

			request: request(),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyScenes",

			request: request(lo.Times(101, func(int) core.ScenesUpdateItem { return scene(uuid.New()) })...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateID",

			request: request(scene(sceneID2), scene(sceneID2)),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(scene(sceneID2)),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(scene(sceneID2)),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(scene(sceneID2), scene(sceneID3)),

			lockMock: &daoMock[bool]{resp: true},
			// One scene is missing, so the update writes none.
			updateMock:   &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 1)}},
			updateScenes: []dao.ScenesUpdateItem{updateScene(sceneID2), updateScene(sceneID3)},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/Update",

			request: request(scene(sceneID2)),

			lockMock:     &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Scene]{err: errFoo},
			updateScenes: []dao.ScenesUpdateItem{updateScene(sceneID2)},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: request(scene(sceneID2)),

			lockMock:     &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 1)}},
			bytesMock:    overBudget,
			updateScenes: []dao.ScenesUpdateItem{updateScene(sceneID2)},

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/ProjectBytes",

			request: request(scene(sceneID2)),

			lockMock:     &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 1)}},
			bytesMock:    &daoMock[*dao.ProjectBytes]{err: errFoo},
			updateScenes: []dao.ScenesUpdateItem{updateScene(sceneID2)},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(scene(sceneID2)),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockScenesUpdateDaoLock(t)
			mockUpdate := coremocks.NewMockScenesUpdateDao(t)
			mockBytes := coremocks.NewMockScenesUpdateDaoBytes(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ScenesUpdateRequest) bool {
						return req.ProjectID == projectID && recent(req.Now) && slices.Equal(req.Scenes, testCase.updateScenes)
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

			service := core.NewScenesUpdate(mockLock, mockUpdate, mockBytes, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
			mockBytes.AssertExpectations(t)
		})
	}
}
