package core_test

import (
	"errors"
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

func TestLayerDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := func(ids ...uuid.UUID) *core.LayerDeleteRequest {
		return &core.LayerDeleteRequest{OwnerID: ownerID, ProjectID: projectID, IDs: ids}
	}

	// kinds is what the kind read returns for layers of one kind, of which the project holds count.
	kinds := func(kind string, count int, ids ...uuid.UUID) []*dao.LayerKind {
		return lo.Map(ids, func(id uuid.UUID, _ int) *dao.LayerKind {
			return &dao.LayerKind{ID: id, Kind: kind, Count: count}
		})
	}

	manuscript := coreLayer(layerID1, 0)
	manuscript.Kind = core.LayerKindManuscript

	testCases := []struct {
		name string

		request *core.LayerDeleteRequest

		lockMock   *daoMock[bool]
		kindMock   *daoMock[[]*dao.LayerKind]
		deleteMock *daoMock[[]*dao.Layer]
		transactor transaction.Transactor

		expect    []*core.Layer
		expectErr error
	}{
		{
			// A project needs no notes layer, so its last one goes.
			name: "Success",

			request: request(layerID2),

			lockMock:   &daoMock[bool]{resp: true},
			kindMock:   &daoMock[[]*dao.LayerKind]{resp: kinds(core.LayerKindNotes, 1, layerID2)},
			deleteMock: &daoMock[[]*dao.Layer]{resp: []*dao.Layer{notesLayer(layerID2, 1)}},

			expect: []*core.Layer{coreLayer(layerID2, 1)},
		},
		{
			name: "Success/OneOfTwoManuscripts",

			request: request(layerID1),

			lockMock: &daoMock[bool]{resp: true},
			kindMock: &daoMock[[]*dao.LayerKind]{resp: kinds(core.LayerKindManuscript, 2, layerID1)},
			deleteMock: &daoMock[[]*dao.Layer]{resp: []*dao.Layer{
				daoLayer(layerID1, 0, core.LayerKindManuscript),
			}},

			expect: []*core.Layer{manuscript},
		},
		{
			name: "Success/Batch",

			request: request(layerID3, layerID1),

			lockMock: &daoMock[bool]{resp: true},
			kindMock: &daoMock[[]*dao.LayerKind]{resp: append(
				kinds(core.LayerKindNotes, 1, layerID3), kinds(core.LayerKindManuscript, 2, layerID1)...,
			)},
			// The delete returns its rows in no particular order.
			deleteMock: &daoMock[[]*dao.Layer]{resp: []*dao.Layer{
				notesLayer(layerID3, 2), daoLayer(layerID1, 0, core.LayerKindManuscript),
			}},

			expect: []*core.Layer{manuscript, coreLayer(layerID3, 2)},
		},
		{
			name: "Error/NoID",

			request: request(),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateID",

			request: request(layerID2, layerID2),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyIDs",

			request: request(lo.Times(101, func(int) uuid.UUID { return uuid.New() })...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(layerID2),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(layerID2),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Kind",

			request: request(layerID2),

			lockMock: &daoMock[bool]{resp: true},
			kindMock: &daoMock[[]*dao.LayerKind]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(layerID2, layerID3),

			lockMock: &daoMock[bool]{resp: true},
			kindMock: &daoMock[[]*dao.LayerKind]{resp: kinds(core.LayerKindNotes, 1, layerID2)},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/OnlyManuscript",

			request: request(layerID1),

			lockMock: &daoMock[bool]{resp: true},
			kindMock: &daoMock[[]*dao.LayerKind]{resp: kinds(core.LayerKindManuscript, 1, layerID1)},

			expectErr: core.ErrRequiredLayer,
		},
		{
			name: "Error/BothManuscripts",

			request: request(layerID1, layerID2),

			lockMock: &daoMock[bool]{resp: true},
			kindMock: &daoMock[[]*dao.LayerKind]{resp: kinds(core.LayerKindManuscript, 2, layerID1, layerID2)},

			expectErr: core.ErrRequiredLayer,
		},
		{
			name: "Error/Delete",

			request: request(layerID2),

			lockMock:   &daoMock[bool]{resp: true},
			kindMock:   &daoMock[[]*dao.LayerKind]{resp: kinds(core.LayerKindNotes, 1, layerID2)},
			deleteMock: &daoMock[[]*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(layerID2),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockLayerDeleteDaoLock(t)
			mockKind := coremocks.NewMockLayerDeleteDaoKind(t)
			mockDelete := coremocks.NewMockLayerDeleteDao(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.kindMock != nil {
				mockKind.EXPECT().
					Exec(mock.Anything, &dao.LayerKindRequest{ProjectID: projectID, IDs: testCase.request.IDs}).
					Return(testCase.kindMock.resp, testCase.kindMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.LayerDeleteRequest{ProjectID: projectID, IDs: testCase.request.IDs}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerDelete(mockLock, mockKind, mockDelete, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockKind.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
		})
	}
}
