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

func TestSceneCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := func(position int) *core.SceneCreateRequest {
		return &core.SceneCreateRequest{
			OwnerID: ownerID, ProjectID: projectID, Position: position,
			Title: "Scene", Color: "#4A90D9", Description: "Paul ment.", Prompt: "Lent.",
		}
	}

	testCases := []struct {
		name string

		request *core.SceneCreateRequest

		lockMock   *daoMock[bool]
		insertMock *daoMock[*dao.Scene]
		transactor transaction.Transactor

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: request(0),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID3, 0)},

			expect: coreScene(sceneID3, 0),
		},
		{
			name: "Success/Position",

			request: request(2),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID3, 2)},

			expect: coreScene(sceneID3, 2),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.SceneCreateRequest{OwnerID: ownerID, ProjectID: projectID, Color: "#4A90D9"},

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
			name: "Error/ProjectFull",

			request: request(0),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Scene]{err: dao.ErrSceneInsertFull},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/Insert",

			request: request(0),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Scene]{err: errFoo},

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

			mockLock := coremocks.NewMockSceneCreateDaoLock(t)
			mockInsert := coremocks.NewMockSceneCreateDaoInsert(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.insertMock != nil {
				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.SceneInsertRequest) bool {
						return req.ID != uuid.Nil && req.ProjectID == projectID && req.Position == testCase.request.Position &&
							req.Title == "Scene" && req.Color == sceneColor && req.Description == "Paul ment." &&
							req.Prompt == "Lent." && recent(req.Now)
					})).
					Return(testCase.insertMock.resp, testCase.insertMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewSceneCreate(mockLock, mockInsert, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
		})
	}
}
