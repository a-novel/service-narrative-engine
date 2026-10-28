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

func TestLayerMove(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := func(after uuid.UUID) *core.LayerMoveRequest {
		return &core.LayerMoveRequest{OwnerID: ownerID, ID: layerID2, After: after}
	}

	testCases := []struct {
		name string

		request *core.LayerMoveRequest

		// targetMock answers both reads of the moved layer, before and after the lock.
		targetMock *daoMock[*dao.Layer]
		lockMock   *daoMock[*dao.Project]
		anchorMock *daoMock[*dao.Layer]
		// to is the position the service moves the layer to, when the case reaches the move.
		to         int
		moveMock   *daoMock[*dao.Layer]
		transactor transaction.Transactor

		expect    *core.Layer
		expectErr error
	}{
		{
			name: "Success/Down",

			request: request(layerID3),

			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},
			to:         2,
			moveMock:   &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 2)},

			expect: coreLayer(layerID2, 2),
		},
		{
			name: "Success/First",

			request: request(uuid.Nil),

			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			to:         0,
			moveMock:   &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 0)},

			expect: coreLayer(layerID2, 0),
		},
		{
			name: "Success/InPlace",

			request: request(layerID1),

			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID1, 0)},

			expect: coreLayer(layerID2, 1),
		},
		{
			name: "Error/Self",

			request: request(layerID2),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest",

			request: &core.LayerMoveRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: request(uuid.Nil),

			targetMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: request(uuid.Nil),

			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: foreignProject()},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/UnknownAnchor",

			request: request(layerID3),

			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/AnchorElsewhere",

			request: request(layerID3),

			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Layer]{resp: func() *dao.Layer {
				anchor := notesLayer(layerID3, 2)
				anchor.ProjectID = otherProjectID

				return anchor
			}()},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Move",

			request: request(uuid.Nil),

			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			to:         0,
			moveMock:   &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(uuid.Nil),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockSelect := coremocks.NewMockLayerMoveDaoSelect(t)
			mockLock := coremocks.NewMockLayerMoveDaoLock(t)
			mockMove := coremocks.NewMockLayerMoveDao(t)

			if testCase.targetMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ID: layerID2}).
					Return(testCase.targetMock.resp, testCase.targetMock.err)
			}

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.anchorMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ID: testCase.request.After}).
					Return(testCase.anchorMock.resp, testCase.anchorMock.err)
			}

			if testCase.moveMock != nil {
				mockMove.EXPECT().
					Exec(mock.Anything, &dao.LayerMoveRequest{ID: layerID2, To: testCase.to}).
					Return(testCase.moveMock.resp, testCase.moveMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerMove(mockSelect, mockLock, mockMove, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockSelect.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockMove.AssertExpectations(t)
		})
	}
}
