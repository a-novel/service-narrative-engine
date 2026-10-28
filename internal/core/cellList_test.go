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

func TestCellList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		request *core.CellListRequest

		projectMock *daoMock[*dao.Project]
		// listRequest is the page the service asks for, when the case reaches the list.
		listRequest *dao.CellListRequest
		listMock    *daoMock[[]*dao.CellListRow]

		expect    *core.CellPage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 50, Offset: 100},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			listRequest: &dao.CellListRequest{ProjectID: projectID, Limit: 50, Offset: 100},
			listMock: &daoMock[[]*dao.CellListRow]{resp: []*dao.CellListRow{
				{Cell: dao.Cell{SceneID: sceneID1, LayerID: layerID1, UpdatedAt: updatedAt}, Total: 120},
			}},

			expect: &core.CellPage{
				Cells: []*core.Cell{{SceneID: sceneID1, LayerID: layerID1, UpdatedAt: updatedAt}},
				Total: 120,
			},
		},
		{
			name: "Success/PastEnd",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			listRequest: &dao.CellListRequest{ProjectID: projectID, Limit: core.CellListDefaultSize},
			listMock:    &daoMock[[]*dao.CellListRow]{},

			expect: &core.CellPage{Cells: []*core.Cell{}},
		},
		{
			name: "Error/LimitTooHigh",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 501},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeOffset",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID, Offset: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{err: dao.ErrProjectSelectNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: foreignProject()},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/List",

			request: &core.CellListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			listRequest: &dao.CellListRequest{ProjectID: projectID, Limit: core.CellListDefaultSize},
			listMock:    &daoMock[[]*dao.CellListRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockProject := coremocks.NewMockCellListDaoProject(t)
			mockList := coremocks.NewMockCellListDao(t)

			if testCase.projectMock != nil {
				mockProject.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID}).
					Return(testCase.projectMock.resp, testCase.projectMock.err)
			}

			if testCase.listMock != nil {
				mockList.EXPECT().
					Exec(mock.Anything, testCase.listRequest).
					Return(testCase.listMock.resp, testCase.listMock.err)
			}

			service := core.NewCellList(mockProject, mockList)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockProject.AssertExpectations(t)
			mockList.AssertExpectations(t)
		})
	}
}
