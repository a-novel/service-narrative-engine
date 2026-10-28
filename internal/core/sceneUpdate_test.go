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

	request := &core.SceneUpdateRequest{OwnerID: ownerID, ID: sceneID2, Title: "Scene", Color: "#4A90D9"}

	testCases := []struct {
		name string

		request *core.SceneUpdateRequest

		selectMock  *daoMock[*dao.Scene]
		projectMock *daoMock[*dao.Project]
		updateMock  *daoMock[*dao.Scene]

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			selectMock:  &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			updateMock:  &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},

			expect: coreScene(sceneID2, 1),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.SceneUpdateRequest{OwnerID: ownerID, ID: sceneID2, Color: "#4A90D9"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: request,

			selectMock: &daoMock[*dao.Scene]{err: dao.ErrSceneSelectNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: request,

			selectMock:  &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			projectMock: &daoMock[*dao.Project]{resp: foreignProject()},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/DeletedMeanwhile",

			request: request,

			selectMock:  &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			updateMock:  &daoMock[*dao.Scene]{err: dao.ErrSceneUpdateNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/Update",

			request: request,

			selectMock:  &daoMock[*dao.Scene]{resp: daoScene(sceneID2, 1)},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			updateMock:  &daoMock[*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockSelect := coremocks.NewMockSceneUpdateDaoSelect(t)
			mockProject := coremocks.NewMockSceneUpdateDaoProject(t)
			mockUpdate := coremocks.NewMockSceneUpdateDao(t)

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.SceneSelectRequest{ID: sceneID2}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			if testCase.projectMock != nil {
				mockProject.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID}).
					Return(testCase.projectMock.resp, testCase.projectMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.SceneUpdateRequest) bool {
						return req.ID == sceneID2 && req.Title == "Scene" && req.Color == sceneColor && recent(req.Now)
					})).
					Return(testCase.updateMock.resp, testCase.updateMock.err)
			}

			service := core.NewSceneUpdate(mockSelect, mockProject, mockUpdate)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockSelect.AssertExpectations(t)
			mockProject.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
		})
	}
}
