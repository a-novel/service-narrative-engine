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

func TestLayerUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.LayerUpdateRequest{OwnerID: ownerID, ProjectID: projectID, ID: layerID2, Name: "Layer"}

	testCases := []struct {
		name string

		request *core.LayerUpdateRequest

		ownerMock  *daoMock[bool]
		updateMock *daoMock[*dao.Layer]

		expect    *core.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},

			expect: coreLayer(layerID2, 1),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.LayerUpdateRequest{OwnerID: ownerID, ProjectID: projectID, ID: layerID2},

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
			updateMock: &daoMock[*dao.Layer]{err: dao.ErrLayerUpdateNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Update",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockLayerUpdateDaoOwner(t)
			mockUpdate := coremocks.NewMockLayerUpdateDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayerUpdateRequest) bool {
						return req.ProjectID == projectID && req.ID == layerID2 && req.Name == "Layer" && recent(req.Now)
					})).
					Return(testCase.updateMock.resp, testCase.updateMock.err)
			}

			service := core.NewLayerUpdate(mockOwner, mockUpdate)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
		})
	}
}
