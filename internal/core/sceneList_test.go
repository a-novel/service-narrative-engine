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

func TestSceneList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		request *core.SceneListRequest

		ownerMock *daoMock[bool]
		// listRequest is the page the service asks for, when the case reaches the list.
		listRequest *dao.SceneListRequest
		listMock    *daoMock[[]*dao.SceneListRow]

		expect    *core.ScenePage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.SceneListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 50, Offset: 100},

			ownerMock:   &daoMock[bool]{resp: true},
			listRequest: &dao.SceneListRequest{ProjectID: projectID, Limit: 50, Offset: 100},
			listMock:    &daoMock[[]*dao.SceneListRow]{resp: []*dao.SceneListRow{{Scene: *daoScene(sceneID1, 100), Total: 120}}},

			expect: &core.ScenePage{Scenes: []*core.Scene{coreScene(sceneID1, 100)}, Total: 120},
		},
		{
			name: "Success/PastEnd",

			request: &core.SceneListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock:   &daoMock[bool]{resp: true},
			listRequest: &dao.SceneListRequest{ProjectID: projectID, Limit: core.SceneListLimitDefault},
			listMock:    &daoMock[[]*dao.SceneListRow]{},

			expect: &core.ScenePage{Scenes: []*core.Scene{}},
		},
		{
			name: "Error/LimitTooHigh",

			request: &core.SceneListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 501},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeLimit",

			request: &core.SceneListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeOffset",

			request: &core.SceneListRequest{OwnerID: ownerID, ProjectID: projectID, Offset: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: &core.SceneListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: &core.SceneListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/List",

			request: &core.SceneListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock:   &daoMock[bool]{resp: true},
			listRequest: &dao.SceneListRequest{ProjectID: projectID, Limit: core.SceneListLimitDefault},
			listMock:    &daoMock[[]*dao.SceneListRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockSceneListDaoOwner(t)
			mockList := coremocks.NewMockSceneListDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.listMock != nil {
				mockList.EXPECT().
					Exec(mock.Anything, testCase.listRequest).
					Return(testCase.listMock.resp, testCase.listMock.err)
			}

			service := core.NewSceneList(mockOwner, mockList)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockList.AssertExpectations(t)
		})
	}
}
