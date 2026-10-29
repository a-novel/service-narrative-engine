package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestSceneUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.SceneUpdateRequest{
		OwnerID: ownerID, ProjectID: projectID, ID: sceneID2, Title: "Scene",
	}

	testCases := []struct {
		name string

		request *core.SceneUpdateRequest

		ownerMock  *daoMock[bool]
		updateMock *daoMock[*dao.Scene]

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},

			expect: coreScene(sceneID2, 1),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.SceneUpdateRequest{OwnerID: ownerID, ProjectID: projectID, ID: sceneID2},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request,

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Scene]{err: dao.ErrSceneUpdateNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/Update",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockSceneUpdateDaoOwner(t)
			mockUpdate := coremocks.NewMockSceneUpdateDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.SceneUpdateRequest) bool {
						return req.ProjectID == projectID && req.ID == sceneID2 &&
							req.Title == "Scene" && recent(req.Now)
					})).
					Return(testCase.updateMock.resp, testCase.updateMock.err)
			}

			service := core.NewSceneUpdate(mockOwner, mockUpdate)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
		})
	}
}
