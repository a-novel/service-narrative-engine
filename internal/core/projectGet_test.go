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

func TestProjectGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		request *core.ProjectGetRequest

		projectMock *daoMock[*dao.Project]
		scenesMock  *daoMock[[]*dao.Scene]
		layersMock  *daoMock[[]*dao.Layer]

		expect    *core.ProjectComposer
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ProjectGetRequest{OwnerID: ownerID, ID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			scenesMock:  &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID1, 0)}},
			layersMock:  &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID2, 0)}},

			expect: &core.ProjectComposer{
				Project: &core.Project{
					ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
					CreatedAt: createdAt, UpdatedAt: updatedAt,
				},
				Scenes: []*core.Scene{
					{ID: sceneID1, ProjectID: projectID, Title: "Scene", Color: sceneColor, UpdatedAt: updatedAt},
				},
				Layers: []*core.Layer{
					{
						ID: layerID2, ProjectID: projectID, Kind: core.LayerKindNotes, Name: "Layer",
						UpdatedAt: updatedAt,
					},
				},
			},
		},
		{
			name: "Error/MissingID",

			request: &core.ProjectGetRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: &core.ProjectGetRequest{OwnerID: ownerID, ID: projectID},

			projectMock: &daoMock[*dao.Project]{err: dao.ErrProjectSelectNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/ListScenes",

			request: &core.ProjectGetRequest{OwnerID: ownerID, ID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			scenesMock:  &daoMock[[]*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ListLayers",

			request: &core.ProjectGetRequest{OwnerID: ownerID, ID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			scenesMock:  &daoMock[[]*dao.Scene]{},
			layersMock:  &daoMock[[]*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoProjectMock := coremocks.NewMockProjectGetDaoProject(t)
			daoScenesMock := coremocks.NewMockProjectGetDaoScenes(t)
			daoLayersMock := coremocks.NewMockProjectGetDaoLayers(t)

			if testCase.projectMock != nil {
				daoProjectMock.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.projectMock.resp, testCase.projectMock.err)
			}

			if testCase.scenesMock != nil {
				daoScenesMock.EXPECT().
					Exec(mock.Anything, &dao.SceneListRequest{ProjectID: projectID}).
					Return(testCase.scenesMock.resp, testCase.scenesMock.err)
			}

			if testCase.layersMock != nil {
				daoLayersMock.EXPECT().
					Exec(mock.Anything, &dao.LayerListRequest{ProjectID: projectID}).
					Return(testCase.layersMock.resp, testCase.layersMock.err)
			}

			service := core.NewProjectGet(daoProjectMock, daoScenesMock, daoLayersMock)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			daoProjectMock.AssertExpectations(t)
			daoScenesMock.AssertExpectations(t)
			daoLayersMock.AssertExpectations(t)
		})
	}
}
