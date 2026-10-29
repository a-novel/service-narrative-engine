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

func TestBeatCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := func(position int) *core.BeatCreateRequest {
		return &core.BeatCreateRequest{
			OwnerID: ownerID, ProjectID: projectID, Position: position,
			Title: "Beat", Color: "#4A90D9", Description: "Paul ment.", Prompt: "Lent.",
		}
	}

	testCases := []struct {
		name string

		request *core.BeatCreateRequest

		lockMock   *daoMock[bool]
		insertMock *daoMock[*dao.Beat]
		transactor transaction.Transactor

		expect    *core.Beat
		expectErr error
	}{
		{
			name: "Success",

			request: request(0),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Beat]{resp: daoBeat(beatID3, 0)},

			expect: coreBeat(beatID3, 0),
		},
		{
			name: "Success/Position",

			request: request(2),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Beat]{resp: daoBeat(beatID3, 2)},

			expect: coreBeat(beatID3, 2),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.BeatCreateRequest{OwnerID: ownerID, ProjectID: projectID, Color: "#4A90D9"},

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

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(0),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectFull",

			request: request(0),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Beat]{err: dao.ErrBeatInsertFull},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/Insert",

			request: request(0),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Beat]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(0),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockBeatCreateDaoLock(t)
			mockInsert := coremocks.NewMockBeatCreateDaoInsert(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.insertMock != nil {
				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.BeatInsertRequest) bool {
						return req.ID != uuid.Nil && req.ProjectID == projectID && req.Position == testCase.request.Position &&
							req.Title == "Beat" && req.Color == beatColor && req.Description == "Paul ment." &&
							req.Prompt == "Lent." && recent(req.Now)
					})).
					Return(testCase.insertMock.resp, testCase.insertMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewBeatCreate(mockLock, mockInsert, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
		})
	}
}
