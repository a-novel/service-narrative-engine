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

	updated := notesLayer(layerID1, 0)
	updated.Description = "Paul ment."
	updated.Revision = 2

	validRequest := &core.LayerUpdateRequest{
		OwnerID: ownerID, ID: layerID1, Revision: 1, Name: "Layer", Description: "Paul ment.",
	}

	testCases := []struct {
		name string

		request *core.LayerUpdateRequest

		daoMock    *daoMock[*dao.Layer]
		selectMock *daoMock[*dao.Layer]

		expect    *core.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: validRequest,

			daoMock: &daoMock[*dao.Layer]{resp: updated},

			expect: &core.Layer{
				ID: layerID1, ProjectID: projectID, Kind: core.LayerKindNotes, Name: "Layer", Description: "Paul ment.",
				Revision:  2,
				UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/BlankName",

			request: &core.LayerUpdateRequest{OwnerID: ownerID, ID: layerID1, Revision: 1, Name: " "},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/RevisionConflict",

			request: validRequest,

			daoMock:    &daoMock[*dao.Layer]{err: dao.ErrLayerUpdateNotFound},
			selectMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID1, 0)},

			expectErr: core.ErrRevisionConflict,
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			daoMock:    &daoMock[*dao.Layer]{err: dao.ErrLayerUpdateNotFound},
			selectMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			daoMock: &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockLayerUpdateDao(t)
			mockSelect := coremocks.NewMockLayerUpdateDaoSelect(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayerUpdateRequest) bool {
						return req.ID == layerID1 && req.OwnerID == ownerID && req.Revision == 1 &&
							req.Description == "Paul ment." && recent(req.Now)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ID: layerID1, OwnerID: ownerID}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			service := core.NewLayerUpdate(mockDao, mockSelect)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockDao.AssertExpectations(t)
			mockSelect.AssertExpectations(t)
		})
	}
}
