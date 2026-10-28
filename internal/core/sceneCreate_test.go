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

func TestSceneCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	scene := core.SceneCreateItem{Title: "Scene", Color: "#4A90D9"}
	// insertScene is scene as the insert receives it, without its generated ID.
	insertScene := dao.SceneInsertItem{Title: "Scene", Color: sceneColor}

	request := func(position int, scenes ...core.SceneCreateItem) *core.SceneCreateRequest {
		return &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Position: position, Scenes: scenes}
	}

	testCases := []struct {
		name string

		request *core.SceneCreateRequest

		ownerMock  *daoMock[bool]
		lockMock   *daoMock[*dao.Project]
		insertMock *daoMock[[]*dao.Scene]
		// insertScenes are the scenes the insert receives, without their generated IDs.
		insertScenes []dao.SceneInsertItem
		transactor   transaction.Transactor

		expect    []*core.Scene
		expectErr error
	}{
		{
			name: "Success/First",

			request: request(0, scene),

			ownerMock:    &daoMock[bool]{resp: true},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			insertMock:   &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID3, 0)}},
			insertScenes: []dao.SceneInsertItem{insertScene},

			expect: []*core.Scene{coreScene(sceneID3, 0)},
		},
		{
			name: "Success/Position",

			request: request(2, scene),

			ownerMock:    &daoMock[bool]{resp: true},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			insertMock:   &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID3, 2)}},
			insertScenes: []dao.SceneInsertItem{insertScene},

			expect: []*core.Scene{coreScene(sceneID3, 2)},
		},
		{
			name: "Success/Batch",

			request: request(1, scene, core.SceneCreateItem{
				Title: "Le thème", Color: "#E0457B", Description: "Paul ment.", Prompt: "Lent.",
			}),

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{resp: daoProject()},
			// The insert returns its rows in no particular order.
			insertMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 2), daoScene(sceneID1, 1)}},
			insertScenes: []dao.SceneInsertItem{
				insertScene,
				{Title: "Le thème", Color: "#e0457b", Description: "Paul ment.", Prompt: "Lent."},
			},

			expect: []*core.Scene{coreScene(sceneID1, 1), coreScene(sceneID2, 2)},
		},
		{
			name: "Error/InvalidRequest",

			request: request(0, core.SceneCreateItem{Color: "#4A90D9"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativePosition",

			request: request(-1, scene),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoScene",

			request: request(0),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyScenes",

			request: request(0, slices.Repeat([]core.SceneCreateItem{scene}, 101)...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(0, scene),

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request(0, scene),

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Lock",

			request: request(0, scene),

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectFull",

			request: request(0, scene),

			ownerMock:    &daoMock[bool]{resp: true},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			insertMock:   &daoMock[[]*dao.Scene]{err: dao.ErrSceneInsertFull},
			insertScenes: []dao.SceneInsertItem{insertScene},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/Insert",

			request: request(0, scene),

			ownerMock:    &daoMock[bool]{resp: true},
			lockMock:     &daoMock[*dao.Project]{resp: daoProject()},
			insertMock:   &daoMock[[]*dao.Scene]{err: errFoo},
			insertScenes: []dao.SceneInsertItem{insertScene},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(0, scene),

			ownerMock:  &daoMock[bool]{resp: true},
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockSceneCreateDaoOwner(t)
			mockLock := coremocks.NewMockSceneCreateDaoLock(t)
			mockInsert := coremocks.NewMockSceneCreateDaoInsert(t)

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

			if testCase.insertMock != nil {
				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.SceneInsertRequest) bool {
						ids := lo.Map(req.Scenes, func(item dao.SceneInsertItem, _ int) uuid.UUID { return item.ID })
						scenes := lo.Map(req.Scenes, func(item dao.SceneInsertItem, _ int) dao.SceneInsertItem {
							item.ID = uuid.Nil

							return item
						})

						return req.ProjectID == projectID && req.Position == testCase.request.Position && recent(req.Now) &&
							freshIDs(ids) && slices.Equal(scenes, testCase.insertScenes)
					})).
					Return(testCase.insertMock.resp, testCase.insertMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewSceneCreate(mockOwner, mockLock, mockInsert, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
		})
	}
}
