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

	request := &core.LayerUpdateRequest{OwnerID: ownerID, ID: layerID2, Name: "Layer"}

	testCases := []struct {
		name string

		request *core.LayerUpdateRequest

		selectMock  *daoMock[*dao.Layer]
		projectMock *daoMock[*dao.Project]
		updateMock  *daoMock[*dao.Layer]

		expect    *core.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			selectMock:  &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			updateMock:  &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},

			expect: coreLayer(layerID2, 1),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.LayerUpdateRequest{OwnerID: ownerID, ID: layerID2},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: request,

			selectMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: request,

			selectMock:  &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			projectMock: &daoMock[*dao.Project]{resp: foreignProject()},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/DeletedMeanwhile",

			request: request,

			selectMock:  &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			updateMock:  &daoMock[*dao.Layer]{err: dao.ErrLayerUpdateNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Update",

			request: request,

			selectMock:  &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			updateMock:  &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockSelect := coremocks.NewMockLayerUpdateDaoSelect(t)
			mockProject := coremocks.NewMockLayerUpdateDaoProject(t)
			mockUpdate := coremocks.NewMockLayerUpdateDao(t)

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ID: layerID2}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			if testCase.projectMock != nil {
				mockProject.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID}).
					Return(testCase.projectMock.resp, testCase.projectMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayerUpdateRequest) bool {
						return req.ID == layerID2 && req.Name == "Layer" && recent(req.Now)
					})).
					Return(testCase.updateMock.resp, testCase.updateMock.err)
			}

			service := core.NewLayerUpdate(mockSelect, mockProject, mockUpdate)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockSelect.AssertExpectations(t)
			mockProject.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
		})
	}
}
