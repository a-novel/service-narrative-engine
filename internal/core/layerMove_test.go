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
		return &core.LayerMoveRequest{OwnerID: ownerID, ProjectID: projectID, ID: layerID2, After: after}
	}

	testCases := []struct {
		name string

		request *core.LayerMoveRequest

		ownerMock  *daoMock[bool]
		lockMock   *daoMock[*dao.Project]
		targetMock *daoMock[*dao.Layer]
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

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			anchorMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},
			to:         2,
			moveMock:   &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 2)},

			expect: coreLayer(layerID2, 2),
		},
		{
			name: "Success/First",

			request: request(uuid.Nil),

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			to:         0,
			moveMock:   &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 0)},

			expect: coreLayer(layerID2, 0),
		},
		{
			name: "Success/InPlace",

			request: request(layerID1),

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
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

			request: &core.LayerMoveRequest{OwnerID: ownerID, ProjectID: projectID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(uuid.Nil),

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request(uuid.Nil),

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectNotFound",

			request: request(uuid.Nil),

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{err: dao.ErrProjectLockNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/NotFound",

			request: request(uuid.Nil),

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			targetMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/UnknownAnchor",

			request: request(layerID3),

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			anchorMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Move",

			request: request(uuid.Nil),

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			targetMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			to:         0,
			moveMock:   &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(uuid.Nil),

			ownerMock:  &daoMock[bool]{resp: true},
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockLayerMoveDaoOwner(t)
			mockLock := coremocks.NewMockLayerMoveDaoLock(t)
			mockSelect := coremocks.NewMockLayerMoveDaoSelect(t)
			mockMove := coremocks.NewMockLayerMoveDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.targetMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ProjectID: projectID, ID: layerID2}).
					Return(testCase.targetMock.resp, testCase.targetMock.err)
			}

			if testCase.anchorMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ProjectID: projectID, ID: testCase.request.After}).
					Return(testCase.anchorMock.resp, testCase.anchorMock.err)
			}

			if testCase.moveMock != nil {
				mockMove.EXPECT().
					Exec(mock.Anything, &dao.LayerMoveRequest{ProjectID: projectID, ID: layerID2, To: testCase.to}).
					Return(testCase.moveMock.resp, testCase.moveMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerMove(mockOwner, mockLock, mockSelect, mockMove, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockSelect.AssertExpectations(t)
			mockMove.AssertExpectations(t)
		})
	}
}
