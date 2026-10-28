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

func TestLayerCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	layer := core.LayerCreateItem{Kind: core.LayerKindNotes, Name: "Layer"}
	// insertLayer is layer as the insert receives it, without its generated ID.
	insertLayer := dao.LayerInsertItem{Kind: core.LayerKindNotes, Name: "Layer"}

	// manuscript is a second manuscript layer, which a project may hold.
	manuscript := coreLayer(layerID1, 1)
	manuscript.Kind = core.LayerKindManuscript

	request := func(position int, layers ...core.LayerCreateItem) *core.LayerCreateRequest {
		return &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Position: position, Layers: layers}
	}

	testCases := []struct {
		name string

		request *core.LayerCreateRequest

		lockMock   *daoMock[bool]
		insertMock *daoMock[[]*dao.Layer]
		// insertLayers are the layers the insert receives, without their generated IDs.
		insertLayers []dao.LayerInsertItem
		transactor   transaction.Transactor

		expect    []*core.Layer
		expectErr error
	}{
		{
			name: "Success/First",

			request: request(0, layer),

			lockMock:     &daoMock[bool]{resp: true},
			insertMock:   &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID3, 0)}},
			insertLayers: []dao.LayerInsertItem{insertLayer},

			expect: []*core.Layer{coreLayer(layerID3, 0)},
		},
		{
			name: "Success/Position",

			request: request(2, layer),

			lockMock:     &daoMock[bool]{resp: true},
			insertMock:   &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID3, 2)}},
			insertLayers: []dao.LayerInsertItem{insertLayer},

			expect: []*core.Layer{coreLayer(layerID3, 2)},
		},
		{
			name: "Success/Batch",

			request: request(1,
				core.LayerCreateItem{Kind: core.LayerKindManuscript, Name: "Layer"},
				core.LayerCreateItem{Kind: core.LayerKindNotes, Name: "Layer", Description: "Who appears."},
			),

			lockMock: &daoMock[bool]{resp: true},
			// The insert returns its rows in no particular order.
			insertMock: &daoMock[[]*dao.Layer]{resp: []*dao.Layer{
				notesLayer(layerID2, 2), daoLayer(layerID1, 1, core.LayerKindManuscript),
			}},
			insertLayers: []dao.LayerInsertItem{
				{Kind: core.LayerKindManuscript, Name: "Layer"},
				{Kind: core.LayerKindNotes, Name: "Layer", Description: "Who appears."},
			},

			expect: []*core.Layer{manuscript, coreLayer(layerID2, 2)},
		},
		{
			name: "Error/InvalidRequest",

			request: request(0, core.LayerCreateItem{Kind: core.LayerKindNotes}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/UnknownKind",

			request: request(0, core.LayerCreateItem{Kind: "chapter", Name: "Layer"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativePosition",

			request: request(-1, layer),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoLayer",

			request: request(0),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyLayers",

			request: request(0, slices.Repeat([]core.LayerCreateItem{layer}, 101)...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(0, layer),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(0, layer),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectFull",

			request: request(0, layer),

			lockMock:     &daoMock[bool]{resp: true},
			insertMock:   &daoMock[[]*dao.Layer]{err: dao.ErrLayerInsertFull},
			insertLayers: []dao.LayerInsertItem{insertLayer},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/Insert",

			request: request(0, layer),

			lockMock:     &daoMock[bool]{resp: true},
			insertMock:   &daoMock[[]*dao.Layer]{err: errFoo},
			insertLayers: []dao.LayerInsertItem{insertLayer},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(0, layer),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockLayerCreateDaoLock(t)
			mockInsert := coremocks.NewMockLayerCreateDaoInsert(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.insertMock != nil {
				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayerInsertRequest) bool {
						ids := lo.Map(req.Layers, func(item dao.LayerInsertItem, _ int) uuid.UUID { return item.ID })
						layers := lo.Map(req.Layers, func(item dao.LayerInsertItem, _ int) dao.LayerInsertItem {
							item.ID = uuid.Nil

							return item
						})

						return req.ProjectID == projectID && req.Position == testCase.request.Position && recent(req.Now) &&
							freshIDs(ids) && slices.Equal(layers, testCase.insertLayers)
					})).
					Return(testCase.insertMock.resp, testCase.insertMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerCreate(mockLock, mockInsert, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
		})
	}
}
