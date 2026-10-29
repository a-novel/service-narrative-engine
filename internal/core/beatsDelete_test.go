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

func TestBeatsDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := func(ids ...uuid.UUID) *core.BeatsDeleteRequest {
		return &core.BeatsDeleteRequest{OwnerID: ownerID, ProjectID: projectID, IDs: ids}
	}

	testCases := []struct {
		name string

		request *core.BeatsDeleteRequest

		lockMock   *daoMock[bool]
		deleteMock *daoMock[[]*dao.Beat]
		transactor transaction.Transactor

		expect    []*core.Beat
		expectErr error
	}{
		{
			name: "Success",

			request: request(beatID2),

			lockMock:   &daoMock[bool]{resp: true},
			deleteMock: &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID2, 1)}},

			expect: []*core.Beat{coreBeat(beatID2, 1)},
		},
		{
			name: "Success/Batch",

			request: request(beatID3, beatID1),

			lockMock: &daoMock[bool]{resp: true},
			// The delete returns its rows in no particular order.
			deleteMock: &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID3, 2), daoBeat(beatID1, 0)}},

			expect: []*core.Beat{coreBeat(beatID1, 0), coreBeat(beatID3, 2)},
		},
		{
			name: "Error/NoID",

			request: request(),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateID",

			request: request(beatID2, beatID2),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyIDs",

			request: request(lo.Times(101, func(int) uuid.UUID { return uuid.New() })...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(beatID2),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(beatID2),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(beatID2, beatID3),

			lockMock: &daoMock[bool]{resp: true},
			// One beat is missing, so the whole delete rolls back.
			deleteMock: &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID2, 1)}},

			expectErr: core.ErrBeatNotFound,
		},
		{
			name: "Error/Delete",

			request: request(beatID2),

			lockMock:   &daoMock[bool]{resp: true},
			deleteMock: &daoMock[[]*dao.Beat]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(beatID2),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockBeatsDeleteDaoLock(t)
			mockDelete := coremocks.NewMockBeatsDeleteDao(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.BeatsDeleteRequest{ProjectID: projectID, IDs: testCase.request.IDs}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewBeatsDelete(mockLock, mockDelete, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
		})
	}
}
