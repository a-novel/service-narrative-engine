package core_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
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

	request := func(id uuid.UUID) *core.LayerDeleteRequest {
		return &core.LayerDeleteRequest{OwnerID: ownerID, ProjectID: projectID, ID: id}
	}

	// kinds is what the kind read returns for a layer of a kind the project holds count layers of.
	kinds := func(id uuid.UUID, kind string, count int) []*dao.LayerKind {
		return []*dao.LayerKind{{ID: id, Kind: kind, Count: count}}
	}

	manuscript := coreLayer(layerID1, 0)
	manuscript.Kind = core.LayerKindManuscript

	testCases := []struct {
		name string

		request *core.LayerDeleteRequest

		lockMock   *daoMock[bool]
		kindMock   *daoMock[[]*dao.LayerKind]
		deleteMock *daoMock[*dao.Layer]
		transactor transaction.Transactor

		expect    *core.Layer
		expectErr error
	}{
		{
			// A project needs no notes layer, so its last one goes.
			name: "Success",

			request: request(layerID2),

			lockMock:   &daoMock[bool]{resp: true},
			kindMock:   &daoMock[[]*dao.LayerKind]{resp: kinds(layerID2, core.LayerKindNotes, 1)},
			deleteMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},

			expect: coreLayer(layerID2, 1),
		},
		{
			name: "Success/OneOfTwoManuscripts",

			request: request(layerID1),

			lockMock:   &daoMock[bool]{resp: true},
			kindMock:   &daoMock[[]*dao.LayerKind]{resp: kinds(layerID1, core.LayerKindManuscript, 2)},
			deleteMock: &daoMock[*dao.Layer]{resp: daoLayer(layerID1, 0, core.LayerKindManuscript)},

			expect: manuscript,
		},
		{
			name: "Error/InvalidRequest",

			request: request(uuid.Nil),

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

			request: request(layerID2),

			lockMock: &daoMock[bool]{resp: true},
			kindMock: &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{}},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/OnlyManuscript",

			request: request(layerID1),

			lockMock: &daoMock[bool]{resp: true},
			kindMock: &daoMock[[]*dao.LayerKind]{resp: kinds(layerID1, core.LayerKindManuscript, 1)},

			expectErr: core.ErrRequiredLayer,
		},
		{
			name: "Error/Delete",

			request: request(layerID2),

			lockMock:   &daoMock[bool]{resp: true},
			kindMock:   &daoMock[[]*dao.LayerKind]{resp: kinds(layerID2, core.LayerKindNotes, 1)},
			deleteMock: &daoMock[*dao.Layer]{err: errFoo},

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
					Exec(mock.Anything, &dao.LayerKindRequest{ProjectID: projectID, IDs: []uuid.UUID{testCase.request.ID}}).
					Return(testCase.kindMock.resp, testCase.kindMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.LayerDeleteRequest{ProjectID: projectID, ID: testCase.request.ID}).
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
