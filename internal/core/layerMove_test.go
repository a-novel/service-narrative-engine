package core_test

import (
	"errors"
	"testing"

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

	request := func(position int) *core.LayerMoveRequest {
		return &core.LayerMoveRequest{OwnerID: ownerID, ProjectID: projectID, ID: layerID2, Position: position}
	}

	testCases := []struct {
		name string

		request *core.LayerMoveRequest

		ownerMock  *daoMock[bool]
		lockMock   *daoMock[*dao.Project]
		moveMock   *daoMock[*dao.Layer]
		transactor transaction.Transactor

		expect    *core.Layer
		expectErr error
	}{
		{
			name: "Success/Down",

			request: request(2),

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{resp: daoProject()},
			moveMock:  &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 2)},

			expect: coreLayer(layerID2, 2),
		},
		{
			name: "Success/First",

			request: request(0),

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{resp: daoProject()},
			moveMock:  &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 0)},

			expect: coreLayer(layerID2, 0),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.LayerMoveRequest{OwnerID: ownerID, ProjectID: projectID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativePosition",

			request: request(-1),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(0),

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request(0),

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Lock",

			request: request(0),

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(0),

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{resp: daoProject()},
			moveMock:  &daoMock[*dao.Layer]{err: dao.ErrLayerMoveNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Move",

			request: request(0),

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{resp: daoProject()},
			moveMock:  &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(0),

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

			if testCase.moveMock != nil {
				mockMove.EXPECT().
					Exec(mock.Anything, &dao.LayerMoveRequest{
						ProjectID: projectID, ID: layerID2, Position: testCase.request.Position,
					}).
					Return(testCase.moveMock.resp, testCase.moveMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerMove(mockOwner, mockLock, mockMove, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockMove.AssertExpectations(t)
		})
	}
}
