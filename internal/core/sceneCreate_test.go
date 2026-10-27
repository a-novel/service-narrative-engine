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

func TestSceneCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	existing := []*dao.Scene{daoScene(sceneID1, 0), daoScene(sceneID2, 1)}

	// The new scene's ID is generated, so a reorder is matched by where the existing scenes land.
	newBetween := func(req *dao.SceneReorderRequest) bool {
		return len(req.IDs) == 3 && req.IDs[0] == sceneID1 && req.IDs[2] == sceneID2
	}
	newFirst := func(req *dao.SceneReorderRequest) bool {
		return len(req.IDs) == 3 && req.IDs[1] == sceneID1 && req.IDs[2] == sceneID2
	}

	testCases := []struct {
		name string

		request    *core.SceneCreateRequest
		transactor transaction.Transactor

		lockMock    *daoMock[*dao.Project]
		listMock    *daoMock[[]*dao.Scene]
		insertMock  *daoMock[[]*dao.Scene]
		reorderMock *daoMock[[]*dao.Scene]
		reorderedBy func(req *dao.SceneReorderRequest) bool

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success/Last",

			request: &core.SceneCreateRequest{
				OwnerID: ownerID, ProjectID: projectID, After: sceneID2, Title: "Scene", Color: "#4A90D9",
			},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			listMock:   &daoMock[[]*dao.Scene]{resp: existing},
			insertMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID3, 2)}},

			expect: &core.Scene{
				ID: sceneID3, ProjectID: projectID, Position: 2, Title: "Scene", Color: sceneColor, UpdatedAt: updatedAt,
			},
		},
		{
			name: "Success/Between",

			request: &core.SceneCreateRequest{
				OwnerID: ownerID, ProjectID: projectID, After: sceneID1, Title: "Scene", Color: "#4A90D9",
			},

			lockMock:    &daoMock[*dao.Project]{resp: daoProject()},
			listMock:    &daoMock[[]*dao.Scene]{resp: existing},
			insertMock:  &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID3, 2)}},
			reorderMock: &daoMock[[]*dao.Scene]{},
			reorderedBy: newBetween,

			expect: &core.Scene{
				ID: sceneID3, ProjectID: projectID, Position: 1, Title: "Scene", Color: sceneColor, UpdatedAt: updatedAt,
			},
		},
		{
			name: "Success/First",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			lockMock:    &daoMock[*dao.Project]{resp: daoProject()},
			listMock:    &daoMock[[]*dao.Scene]{resp: existing},
			insertMock:  &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID3, 2)}},
			reorderMock: &daoMock[[]*dao.Scene]{},
			reorderedBy: newFirst,

			expect: &core.Scene{
				ID: sceneID3, ProjectID: projectID, Position: 0, Title: "Scene", Color: sceneColor, UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/MissingProject",

			request: &core.SceneCreateRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			lockMock: &daoMock[*dao.Project]{err: dao.ErrProjectLockNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/ProjectFull",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			lockMock: &daoMock[*dao.Project]{resp: daoProject()},
			listMock: &daoMock[[]*dao.Scene]{resp: append(existing, daoScene(sceneID3, 2))},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/UnknownAnchor",

			request: &core.SceneCreateRequest{
				OwnerID: ownerID, ProjectID: projectID, After: sceneID3, Title: "Scene", Color: "#4A90D9",
			},

			lockMock: &daoMock[*dao.Project]{resp: daoProject()},
			listMock: &daoMock[[]*dao.Scene]{resp: existing},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Insert",

			request: &core.SceneCreateRequest{
				OwnerID: ownerID, ProjectID: projectID, After: sceneID2, Title: "Scene", Color: "#4A90D9",
			},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			listMock:   &daoMock[[]*dao.Scene]{resp: existing},
			insertMock: &daoMock[[]*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Reorder",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			lockMock:    &daoMock[*dao.Project]{resp: daoProject()},
			listMock:    &daoMock[[]*dao.Scene]{resp: existing},
			insertMock:  &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID3, 2)}},
			reorderMock: &daoMock[[]*dao.Scene]{err: errFoo},
			reorderedBy: newFirst,

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request:    &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockSceneCreateDaoLock(t)
			mockList := coremocks.NewMockSceneCreateDaoList(t)
			mockInsert := coremocks.NewMockSceneCreateDaoInsert(t)
			mockReorder := coremocks.NewMockSceneCreateDaoReorder(t)

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

			if testCase.insertMock != nil {
				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.SceneInsertRequest) bool {
						return req.ProjectID == projectID && req.Position == len(existing) &&
							len(req.Scenes) == 1 && req.Scenes[0].Title == testCase.request.Title &&
							req.Scenes[0].Color == sceneColor && recent(req.Now)
					})).
					Return(testCase.insertMock.resp, testCase.insertMock.err)
			}

			if testCase.reorderMock != nil {
				mockReorder.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(testCase.reorderedBy)).
					Return(testCase.reorderMock.resp, testCase.reorderMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewSceneCreate(mockLock, mockList, mockInsert, mockReorder, transactor, testLimits)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockList.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
			mockReorder.AssertExpectations(t)
		})
	}
}
