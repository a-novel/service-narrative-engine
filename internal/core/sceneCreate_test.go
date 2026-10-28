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

	testCases := []struct {
		name string

		request *core.SceneCreateRequest

		ownerMock  *daoMock[bool]
		lockMock   *daoMock[*dao.Project]
		insertMock *daoMock[[]*dao.Scene]
		transactor transaction.Transactor

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success/First",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID3, 0)}},

			expect: coreScene(sceneID3, 0),
		},
		{
			name: "Success/Position",

			request: &core.SceneCreateRequest{
				OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9", Position: 2,
			},

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID3, 2)}},

			expect: coreScene(sceneID3, 2),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Color: "#4A90D9"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativePosition",

			request: &core.SceneCreateRequest{
				OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9", Position: -1,
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Lock",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectFull",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[[]*dao.Scene]{err: dao.ErrSceneInsertFull},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/Insert",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[[]*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

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
						return req.ProjectID == projectID && req.Position == testCase.request.Position &&
							len(req.Scenes) == 1 && req.Scenes[0].Title == "Scene" && req.Scenes[0].Color == sceneColor && recent(req.Now)
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
