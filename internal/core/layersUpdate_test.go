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

		lockMock   *daoMock[bool]
		updateMock *daoMock[[]*dao.Layer]
		bytesMock  *daoMock[*dao.ProjectBytes]
		transactor transaction.Transactor
		// updateLayers are the layers the update receives.
		updateLayers []dao.LayersUpdateItem

		expect    []*core.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: request(layer(layerID2)),

			lockMock:     &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID2, 1)}},
			bytesMock:    withinBudget,
			updateLayers: []dao.LayersUpdateItem{updateLayer(layerID2)},

			expect: []*core.Layer{coreLayer(layerID2, 1)},
		},
		{
			name: "Success/Batch",

			request: request(layer(layerID3), core.LayersUpdateItem{
				ID: layerID1, Name: "Personnages", Description: "Who appears.",
			}),

			lockMock: &daoMock[bool]{resp: true},
			// The update returns its rows in no particular order.
			updateMock: &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID1, 0), notesLayer(layerID3, 2)}},
			bytesMock:  withinBudget,
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

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(layer(layerID2)),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(layer(layerID2), layer(layerID3)),

			lockMock: &daoMock[bool]{resp: true},
			// One layer is missing, so the update writes none.
			updateMock:   &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID2, 1)}},
			updateLayers: []dao.LayersUpdateItem{updateLayer(layerID2), updateLayer(layerID3)},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Update",

			request: request(layer(layerID2)),

			lockMock:     &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Layer]{err: errFoo},
			updateLayers: []dao.LayersUpdateItem{updateLayer(layerID2)},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: request(layer(layerID2)),

			lockMock:     &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID2, 1)}},
			bytesMock:    overBudget,
			updateLayers: []dao.LayersUpdateItem{updateLayer(layerID2)},

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/ProjectBytes",

			request: request(layer(layerID2)),

			lockMock:     &daoMock[bool]{resp: true},
			updateMock:   &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID2, 1)}},
			bytesMock:    &daoMock[*dao.ProjectBytes]{err: errFoo},
			updateLayers: []dao.LayersUpdateItem{updateLayer(layerID2)},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(layer(layerID2)),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockLayersUpdateDaoLock(t)
			mockUpdate := coremocks.NewMockLayersUpdateDao(t)
			mockBytes := coremocks.NewMockLayersUpdateDaoBytes(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayersUpdateRequest) bool {
						return req.ProjectID == projectID && recent(req.Now) && slices.Equal(req.Layers, testCase.updateLayers)
					})).
					Return(testCase.updateMock.resp, testCase.updateMock.err)
			}

			if testCase.bytesMock != nil {
				mockBytes.EXPECT().
					Exec(mock.Anything, &dao.ProjectBytesRequest{ID: projectID}).
					Return(testCase.bytesMock.resp, testCase.bytesMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayersUpdate(mockLock, mockUpdate, mockBytes, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
			mockBytes.AssertExpectations(t)
		})
	}
}
