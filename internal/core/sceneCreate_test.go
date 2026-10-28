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

		lockMock   *daoMock[*dao.Project]
		anchorMock *daoMock[*dao.Scene]
		// position is where the service inserts the scene, when the case reaches the insert.
		position   int
		insertMock *daoMock[[]*dao.Scene]
		transactor transaction.Transactor

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success/First",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			position:   0,
			insertMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID3, 0)}},

			expect: coreScene(sceneID3, 0),
		},
		{
			name: "Success/AfterAnchor",

			request: &core.SceneCreateRequest{
				OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9", After: sceneID2,
			},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			position:   2,
			insertMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID3, 2)}},

			expect: coreScene(sceneID3, 2),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Color: "#4A90D9"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			lockMock: &daoMock[*dao.Project]{err: dao.ErrProjectLockNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			lockMock: &daoMock[*dao.Project]{resp: foreignProject()},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/UnknownAnchor",

			request: &core.SceneCreateRequest{
				OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9", After: sceneID2,
			},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Scene]{err: dao.ErrSceneSelectNotFound},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/AnchorElsewhere",

			request: &core.SceneCreateRequest{
				OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9", After: sceneID2,
			},

			lockMock: &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Scene]{resp: func() *dao.Scene {
				anchor := daoScene(sceneID3, 2)
				anchor.ProjectID = otherProjectID

				return anchor
			}()},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectFull",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[[]*dao.Scene]{err: dao.ErrSceneInsertFull},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/Insert",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[[]*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Title: "Scene", Color: "#4A90D9"},

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockSceneCreateDaoLock(t)
			mockSelect := coremocks.NewMockSceneCreateDaoSelect(t)
			mockInsert := coremocks.NewMockSceneCreateDaoInsert(t)

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

			if testCase.insertMock != nil {
				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.SceneInsertRequest) bool {
						return req.ProjectID == projectID && req.Position == testCase.position &&
							len(req.Scenes) == 1 && req.Scenes[0].Title == "Scene" && req.Scenes[0].Color == sceneColor && recent(req.Now)
					})).
					Return(testCase.insertMock.resp, testCase.insertMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewSceneCreate(mockLock, mockSelect, mockInsert, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockSelect.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
		})
	}
}
