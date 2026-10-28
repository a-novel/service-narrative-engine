package core_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestLayersUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	layer := func(id uuid.UUID) core.LayersUpdateItem {
		return core.LayersUpdateItem{ID: id, Name: "Layer"}
	}

	// updateLayer is layer as the update receives it.
	updateLayer := func(id uuid.UUID) dao.LayersUpdateItem {
		return dao.LayersUpdateItem{ID: id, Name: "Layer"}
	}

	request := func(layers ...core.LayersUpdateItem) *core.LayersUpdateRequest {
		return &core.LayersUpdateRequest{OwnerID: ownerID, ProjectID: projectID, Layers: layers}
	}

	testCases := []struct {
		name string

		request *core.LayersUpdateRequest

		ownerMock  *daoMock[bool]
		updateMock *daoMock[[]*dao.Layer]
		// updateLayers are the layers the update receives.
		updateLayers []dao.LayersUpdateItem

		expect    []*core.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: request(layer(layerID2)),

			ownerMock:    &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID2, 1)}},
			updateLayers: []dao.LayersUpdateItem{updateLayer(layerID2)},

			expect: []*core.Layer{coreLayer(layerID2, 1)},
		},
		{
			name: "Success/Batch",

			request: request(layer(layerID3), core.LayersUpdateItem{
				ID: layerID1, Name: "Personnages", Description: "Who appears.",
			}),

			ownerMock: &daoMock[bool]{resp: true},
			// The update returns its rows in no particular order.
			updateMock: &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID1, 0), notesLayer(layerID3, 2)}},
			updateLayers: []dao.LayersUpdateItem{
				updateLayer(layerID3),
				{ID: layerID1, Name: "Personnages", Description: "Who appears."},
			},

			expect: []*core.Layer{coreLayer(layerID3, 2), coreLayer(layerID1, 0)},
		},
		{
			name: "Error/InvalidRequest",

			request: request(core.LayersUpdateItem{ID: layerID2}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoLayer",

			request: request(),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyLayers",

			request: request(lo.Times(101, func(int) core.LayersUpdateItem { return layer(uuid.New()) })...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateID",

			request: request(layer(layerID2), layer(layerID2)),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(layer(layerID2)),

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request(layer(layerID2)),

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(layer(layerID2), layer(layerID3)),

			ownerMock: &daoMock[bool]{resp: true},
			// One layer is missing, so the update writes none.
			updateMock:   &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID2, 1)}},
			updateLayers: []dao.LayersUpdateItem{updateLayer(layerID2), updateLayer(layerID3)},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Update",

			request: request(layer(layerID2)),

			ownerMock:    &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Layer]{err: errFoo},
			updateLayers: []dao.LayersUpdateItem{updateLayer(layerID2)},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockLayersUpdateDaoOwner(t)
			mockUpdate := coremocks.NewMockLayersUpdateDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayersUpdateRequest) bool {
						return req.ProjectID == projectID && recent(req.Now) && slices.Equal(req.Layers, testCase.updateLayers)
					})).
					Return(testCase.updateMock.resp, testCase.updateMock.err)
			}

			service := core.NewLayersUpdate(mockOwner, mockUpdate)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
		})
	}
}
