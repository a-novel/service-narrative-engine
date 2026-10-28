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

		ownerMock *daoMock[bool]
		// listRequest is the page the service asks for, when the case reaches the list.
		listRequest *dao.LayerListRequest
		listMock    *daoMock[[]*dao.LayerListRow]

		expect    *core.LayerPage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 50, Offset: 100},

			ownerMock:   &daoMock[bool]{resp: true},
			listRequest: &dao.LayerListRequest{ProjectID: projectID, Limit: 50, Offset: 100},
			listMock: &daoMock[[]*dao.LayerListRow]{resp: []*dao.LayerListRow{
				{Layer: *notesLayer(layerID1, 100), Total: 120},
			}},

			expect: &core.LayerPage{Layers: []*core.Layer{coreLayer(layerID1, 100)}, Total: 120},
		},
		{
			name: "Success/PastEnd",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock:   &daoMock[bool]{resp: true},
			listRequest: &dao.LayerListRequest{ProjectID: projectID, Limit: core.LayerListLimitDefault},
			listMock:    &daoMock[[]*dao.LayerListRow]{},

			expect: &core.LayerPage{Layers: []*core.Layer{}},
		},
		{
			name: "Error/LimitTooHigh",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 501},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeLimit",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeOffset",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID, Offset: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/List",

			request: &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock:   &daoMock[bool]{resp: true},
			listRequest: &dao.LayerListRequest{ProjectID: projectID, Limit: core.LayerListLimitDefault},
			listMock:    &daoMock[[]*dao.LayerListRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockLayerListDaoOwner(t)
			mockList := coremocks.NewMockLayerListDao(t)

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

			service := core.NewLayerList(mockOwner, mockList)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockList.AssertExpectations(t)
		})
	}
}
