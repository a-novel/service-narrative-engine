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

func TestProjectCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	manuscript := daoLayer(layerID1, 0, core.LayerKindManuscript)
	manuscript.Name = "Manuscript"

	validRequest := &core.ProjectCreateRequest{
		OwnerID:  ownerID,
		Title:    "Les Heures claires",
		Language: "fr",
		Scenes: []core.ProjectCreateScene{
			{Title: "Ouverture", Color: "#4A90D9"},
			{Title: "Le thème", Color: "#e0457b", Description: "Paul ment.", Prompt: "Lent."},
		},
	}

	testCases := []struct {
		name string

		request    *core.ProjectCreateRequest
		transactor transaction.Transactor

		projectMock *daoMock[*dao.Project]
		layerMock   *daoMock[*dao.Layer]
		scenesMock  *daoMock[[]*dao.Scene]

		expect    *core.ProjectComposer
		expectErr error
	}{
		{
			name: "Success",

			request: validRequest,

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layerMock:   &daoMock[*dao.Layer]{resp: manuscript},
			// The insert returns its rows in no particular order.
			scenesMock: &daoMock[[]*dao.Scene]{resp: []*dao.Scene{daoScene(sceneID2, 1), daoScene(sceneID1, 0)}},

			expect: &core.ProjectComposer{
				Project: &core.Project{
					ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
					CreatedAt: createdAt, UpdatedAt: updatedAt,
				},
				Scenes: []*core.Scene{
					{ID: sceneID1, ProjectID: projectID, Position: 0, Title: "Scene", Color: sceneColor, UpdatedAt: updatedAt},
					{ID: sceneID2, ProjectID: projectID, Position: 1, Title: "Scene", Color: sceneColor, UpdatedAt: updatedAt},
				},
				Layers: []*core.Layer{
					{
						ID: layerID1, ProjectID: projectID, Kind: core.LayerKindManuscript, Name: "Manuscript",
						UpdatedAt: updatedAt,
					},
				},
			},
		},
		{
			name: "Success/NoScene",

			request: &core.ProjectCreateRequest{OwnerID: ownerID, Title: "Les Heures claires", Language: "fr"},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layerMock:   &daoMock[*dao.Layer]{resp: manuscript},

			expect: &core.ProjectComposer{
				Project: &core.Project{
					ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
					CreatedAt: createdAt, UpdatedAt: updatedAt,
				},
				Layers: []*core.Layer{
					{
						ID: layerID1, ProjectID: projectID, Kind: core.LayerKindManuscript, Name: "Manuscript",
						UpdatedAt: updatedAt,
					},
				},
			},
		},
		{
			name: "Error/BlankTitle",

			request: &core.ProjectCreateRequest{OwnerID: ownerID, Title: "   ", Language: "fr"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/UnknownLanguage",

			request: &core.ProjectCreateRequest{OwnerID: ownerID, Title: "Clear Hours", Language: "de"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyScenes",

			request: &core.ProjectCreateRequest{
				OwnerID: ownerID, Title: "Clear Hours", Language: "en",
				Scenes: []core.ProjectCreateScene{
					{Title: "Un", Color: sceneColor}, {Title: "Deux", Color: sceneColor}, {Title: "Trois", Color: sceneColor},
				},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InsertProject",

			request: validRequest,

			projectMock: &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/InsertLayer",

			request: validRequest,

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layerMock:   &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/InsertScenes",

			request: validRequest,

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layerMock:   &daoMock[*dao.Layer]{resp: manuscript},
			scenesMock:  &daoMock[[]*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request:    validRequest,
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoProjectMock := coremocks.NewMockProjectCreateDaoProject(t)
			daoLayerMock := coremocks.NewMockProjectCreateDaoLayer(t)
			daoScenesMock := coremocks.NewMockProjectCreateDaoScenes(t)

			if testCase.projectMock != nil {
				daoProjectMock.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectInsertRequest) bool {
						return req.OwnerID == testCase.request.OwnerID && req.Title == testCase.request.Title &&
							req.Language == testCase.request.Language && recent(req.Now)
					})).
					Return(testCase.projectMock.resp, testCase.projectMock.err)
			}

			if testCase.layerMock != nil {
				daoLayerMock.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayerInsertRequest) bool {
						return req.ProjectID == projectID && req.Kind == core.LayerKindManuscript && req.Position == 0
					})).
					Return(testCase.layerMock.resp, testCase.layerMock.err)
			}

			if testCase.scenesMock != nil {
				daoScenesMock.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.SceneInsertRequest) bool {
						return req.ProjectID == projectID && req.Position == 0 &&
							len(req.Scenes) == len(testCase.request.Scenes) &&
							req.Scenes[0].Color == sceneColor &&
							req.Scenes[1].Description == testCase.request.Scenes[1].Description
					})).
					Return(testCase.scenesMock.resp, testCase.scenesMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewProjectCreate(daoProjectMock, daoLayerMock, daoScenesMock, transactor, testLimits)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			daoProjectMock.AssertExpectations(t)
			daoLayerMock.AssertExpectations(t)
			daoScenesMock.AssertExpectations(t)
		})
	}
}
