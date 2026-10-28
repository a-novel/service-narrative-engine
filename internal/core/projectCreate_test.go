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

func TestProjectCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	manuscript := daoLayer(layerID1, 0, core.LayerKindManuscript)
	manuscript.Name = "Manuscript"

	request := func(title, language string) *core.ProjectCreateRequest {
		return &core.ProjectCreateRequest{OwnerID: ownerID, Title: title, Premise: "Paul ment.", Language: language}
	}

	testCases := []struct {
		name string

		request    *core.ProjectCreateRequest
		transactor transaction.Transactor

		projectMock *daoMock[*dao.Project]
		layersMock  *daoMock[[]*dao.Layer]

		expect    *core.ProjectComposer
		expectErr error
	}{
		{
			name: "Success",

			request: request("Les Heures claires", "fr"),

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layersMock:  &daoMock[[]*dao.Layer]{resp: []*dao.Layer{manuscript}},

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

			request: request("   ", "fr"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/UnknownLanguage",

			request: request("Clear Hours", "de"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InsertProject",

			request: request("Les Heures claires", "fr"),

			projectMock: &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/InsertLayers",

			request: request("Les Heures claires", "fr"),

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layersMock:  &daoMock[[]*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request:    request("Les Heures claires", "fr"),
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoProjectMock := coremocks.NewMockProjectCreateDaoProject(t)
			daoLayersMock := coremocks.NewMockProjectCreateDaoLayers(t)

			if testCase.projectMock != nil {
				daoProjectMock.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectInsertRequest) bool {
						return req.ID != uuid.Nil && req.OwnerID == testCase.request.OwnerID &&
							req.Title == testCase.request.Title && req.Premise == testCase.request.Premise &&
							req.Language == testCase.request.Language && recent(req.Now)
					})).
					Return(testCase.projectMock.resp, testCase.projectMock.err)
			}

			if testCase.layersMock != nil {
				daoLayersMock.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayerInsertRequest) bool {
						ids := lo.Map(req.Layers, func(item dao.LayerInsertItem, _ int) uuid.UUID { return item.ID })
						layers := lo.Map(req.Layers, func(item dao.LayerInsertItem, _ int) dao.LayerInsertItem {
							item.ID = uuid.Nil

							return item
						})

						return req.ProjectID == projectID && req.Position == 0 && recent(req.Now) && freshIDs(ids) &&
							slices.Equal(layers, []dao.LayerInsertItem{{Kind: core.LayerKindManuscript, Name: "Manuscript"}})
					})).
					Return(testCase.layersMock.resp, testCase.layersMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewProjectCreate(daoProjectMock, daoLayersMock, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			daoProjectMock.AssertExpectations(t)
			daoLayersMock.AssertExpectations(t)
		})
	}
}
