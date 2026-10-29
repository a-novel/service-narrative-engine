package core_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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

		ownerMock  *daoMock[bool]
		updateMock *daoMock[[]*dao.Scene]
		// updateScenes are the scenes the update receives.
		updateScenes []dao.ScenesUpdateItem

		expect    []*core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: request(scene(sceneID2)),

			ownerMock:    &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 1)}},
			updateScenes: []dao.ScenesUpdateItem{updateScene(sceneID2)},

			expect: []*core.Scene{coreScene(sceneID2, 1)},
		},
		{
			name: "Success/Batch",

			request: request(scene(sceneID3), core.ScenesUpdateItem{
				ID: sceneID1, Title: "Le thème", Description: "Paul ment.", Prompt: "Lent.",
			}),

			ownerMock: &daoMock[bool]{resp: true},
			// The update returns its rows in no particular order.
			updateMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID1, 0), daoScene(sceneID3, 2)}},
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

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request(scene(sceneID2)),

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(scene(sceneID2), scene(sceneID3)),

			ownerMock: &daoMock[bool]{resp: true},
			// One scene is missing, so the update writes none.
			updateMock:   &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 1)}},
			updateScenes: []dao.ScenesUpdateItem{updateScene(sceneID2), updateScene(sceneID3)},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/Update",

			request: request(scene(sceneID2)),

			ownerMock:    &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Scene]{err: errFoo},
			updateScenes: []dao.ScenesUpdateItem{updateScene(sceneID2)},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockScenesUpdateDaoOwner(t)
			mockUpdate := coremocks.NewMockScenesUpdateDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ScenesUpdateRequest) bool {
						return req.ProjectID == projectID && recent(req.Now) && slices.Equal(req.Scenes, testCase.updateScenes)
					})).
					Return(testCase.updateMock.resp, testCase.updateMock.err)
			}

			service := core.NewScenesUpdate(mockOwner, mockUpdate)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
		})
	}
}
