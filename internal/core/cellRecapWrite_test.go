package core_test

import (
	"encoding/json"
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

// recapOutput is a consolidate output, as a recap write takes it.
var recapOutput = json.RawMessage(
	`{"condensed":"Marie attend.","compact":"Marie attend.","essentials":"Marie attend.","names":["Marie"]}`,
)

func TestCellRecapWrite(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := func(recap json.RawMessage) *core.CellRecapWriteRequest {
		return &core.CellRecapWriteRequest{
			OwnerID: ownerID, ProjectID: projectID, SceneID: sceneID1, LayerID: layerID1,
			Recap: recap, CellUpdatedAt: updatedAt,
		}
	}

	testCases := []struct {
		name string

		request *core.CellRecapWriteRequest

		lockMock   *daoMock[bool]
		daoMock    *daoMock[struct{}]
		bytesMock  *daoMock[*dao.ProjectBytes]
		transactor transaction.Transactor

		expectErr error
	}{
		{
			name: "Success",

			request: request(recapOutput),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{},
			bytesMock: withinBudget,
		},
		{
			name: "Error/NoStamp",

			request: &core.CellRecapWriteRequest{
				OwnerID: ownerID, ProjectID: projectID, SceneID: sceneID1, LayerID: layerID1, Recap: recapOutput,
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MalformedRecap",

			request: request(json.RawMessage(`{"condensed":"Marie attend."}`)),

			expectErr: core.ErrInvalidRequest,
		},
		{
			// A project of another user reads as absent, and nothing is written.
			name: "Error/OtherOwner",

			request: request(recapOutput),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CellNotFound",

			request: request(recapOutput),

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[struct{}]{err: dao.ErrCellRecapUpdateNotFound},

			expectErr: core.ErrCellNotFound,
		},
		{
			name: "Error/Conflict",

			request: request(recapOutput),

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[struct{}]{err: dao.ErrCellRecapUpdateConflict},

			expectErr: core.ErrRecapConflict,
		},
		{
			name: "Error/Owner",

			request: request(recapOutput),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Dao",

			request: request(recapOutput),

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[struct{}]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: request(recapOutput),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{},
			bytesMock: overBudget,

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/ProjectBytes",

			request: request(recapOutput),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{},
			bytesMock: &daoMock[*dao.ProjectBytes]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(recapOutput),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockCellRecapWriteDaoLock(t)
			mockDao := coremocks.NewMockCellRecapWriteDao(t)
			mockBytes := coremocks.NewMockCellRecapWriteDaoBytes(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CellRecapUpdateRequest{
						ProjectID: projectID, SceneID: sceneID1, LayerID: layerID1, Recap: recapOutput,
						CellUpdatedAt: updatedAt,
					}).
					Return(testCase.daoMock.err)
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

			service := core.NewCellRecapWrite(mockLock, mockDao, mockBytes, transactor)

			err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			mockLock.AssertExpectations(t)
			mockDao.AssertExpectations(t)
			mockBytes.AssertExpectations(t)
		})
	}
}
