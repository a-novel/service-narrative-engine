package core_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestCellList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	content := json.RawMessage(`{"text":"Marie"}`)
	cell := &dao.Cell{SceneID: sceneID1, LayerID: layerID2, Content: content, UpdatedAt: updatedAt}

	testCases := []struct {
		name string

		request *core.CellListRequest

		projectMock *daoMock[*dao.Project]
		// listRequest is the page the service asks for, when the case reaches the list.
		listRequest *dao.CellListRequest
		listMock    *daoMock[[]*dao.Cell]
		countMock   *daoMock[*dao.CellCount]

		expect    *core.CellPage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.CellListRequest{
				OwnerID: ownerID, ProjectID: projectID, Limit: 50, AfterSceneID: sceneID1, AfterLayerID: layerID1,
			},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			listRequest: &dao.CellListRequest{
				ProjectID: projectID, Limit: 50, AfterSceneID: sceneID1, AfterLayerID: layerID1,
			},
			listMock:  &daoMock[[]*dao.Cell]{resp: []*dao.Cell{cell}},
			countMock: &daoMock[*dao.CellCount]{resp: &dao.CellCount{Count: 2}},

			expect: &core.CellPage{
				Cells: []*core.Cell{{SceneID: sceneID1, LayerID: layerID2, Content: content, UpdatedAt: updatedAt}},
				Total: 2,
			},
		},
		{
			name: "Success/DefaultLimit",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			listRequest: &dao.CellListRequest{ProjectID: projectID, Limit: core.CellListDefaultSize},
			listMock:    &daoMock[[]*dao.Cell]{},
			countMock:   &daoMock[*dao.CellCount]{resp: &dao.CellCount{}},

			expect: &core.CellPage{Cells: []*core.Cell{}},
		},
		{
			name: "Error/LimitTooHigh",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 501},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MissingProjectID",

			request: &core.CellListRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{err: dao.ErrProjectSelectNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/List",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			listRequest: &dao.CellListRequest{ProjectID: projectID, Limit: core.CellListDefaultSize},
			listMock:    &daoMock[[]*dao.Cell]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Count",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			listRequest: &dao.CellListRequest{ProjectID: projectID, Limit: core.CellListDefaultSize},
			listMock:    &daoMock[[]*dao.Cell]{},
			countMock:   &daoMock[*dao.CellCount]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoProjectMock := coremocks.NewMockCellListDaoProject(t)
			daoListMock := coremocks.NewMockCellListDao(t)
			daoCountMock := coremocks.NewMockCellListDaoCount(t)

			if testCase.projectMock != nil {
				daoProjectMock.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.projectMock.resp, testCase.projectMock.err)
			}

			if testCase.listMock != nil {
				daoListMock.EXPECT().
					Exec(mock.Anything, testCase.listRequest).
					Return(testCase.listMock.resp, testCase.listMock.err)
			}

			if testCase.countMock != nil {
				daoCountMock.EXPECT().
					Exec(mock.Anything, &dao.CellCountRequest{ProjectID: projectID}).
					Return(testCase.countMock.resp, testCase.countMock.err)
			}

			service := core.NewCellList(daoProjectMock, daoListMock, daoCountMock)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			daoProjectMock.AssertExpectations(t)
			daoListMock.AssertExpectations(t)
			daoCountMock.AssertExpectations(t)
		})
	}
}
