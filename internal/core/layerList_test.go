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

func TestLayerList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		request *core.LayerListRequest

		projectMock *daoMock[*dao.Project]
		// listRequest is the page the service asks for, when the case reaches the list.
		listRequest *dao.LayerListRequest
		listMock    *daoMock[[]*dao.LayerListRow]

		expect    *core.LayerPage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 50, Offset: 100},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			listRequest: &dao.LayerListRequest{ProjectID: projectID, Limit: 50, Offset: 100},
			listMock: &daoMock[[]*dao.LayerListRow]{resp: []*dao.LayerListRow{
				{Layer: *notesLayer(layerID1, 100), Total: 120},
			}},

			expect: &core.LayerPage{Layers: []*core.Layer{coreLayer(layerID1, 100)}, Total: 120},
		},
		{
			name: "Success/PastEnd",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			listRequest: &dao.LayerListRequest{ProjectID: projectID, Limit: core.LayerListDefaultSize},
			listMock:    &daoMock[[]*dao.LayerListRow]{},

			expect: &core.LayerPage{Layers: []*core.Layer{}},
		},
		{
			name: "Error/LimitTooHigh",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 501},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeOffset",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID, Offset: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{err: dao.ErrProjectSelectNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: foreignProject()},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/List",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID},

			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			listRequest: &dao.LayerListRequest{ProjectID: projectID, Limit: core.LayerListDefaultSize},
			listMock:    &daoMock[[]*dao.LayerListRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockProject := coremocks.NewMockLayerListDaoProject(t)
			mockList := coremocks.NewMockLayerListDao(t)

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

			service := core.NewLayerList(mockProject, mockList)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockProject.AssertExpectations(t)
			mockList.AssertExpectations(t)
		})
	}
}
