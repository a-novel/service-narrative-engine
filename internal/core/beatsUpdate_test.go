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

func TestBeatsUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	beat := func(id uuid.UUID) core.BeatsUpdateItem {
		return core.BeatsUpdateItem{ID: id, Title: "Beat", Color: "#4A90D9"}
	}

	// updateBeat is beat as the update receives it.
	updateBeat := func(id uuid.UUID) dao.BeatsUpdateItem {
		return dao.BeatsUpdateItem{ID: id, Title: "Beat", Color: beatColor}
	}

	request := func(beats ...core.BeatsUpdateItem) *core.BeatsUpdateRequest {
		return &core.BeatsUpdateRequest{OwnerID: ownerID, ProjectID: projectID, Beats: beats}
	}

	testCases := []struct {
		name string

		request *core.BeatsUpdateRequest

		lockMock   *daoMock[bool]
		updateMock *daoMock[[]*dao.Beat]
		bytesMock  *daoMock[*dao.ProjectBytes]
		transactor transaction.Transactor
		// updateBeats are the beats the update receives.
		updateBeats []dao.BeatsUpdateItem

		expect    []*core.Beat
		expectErr error
	}{
		{
			name: "Success",

			request: request(beat(beatID2)),

			lockMock:    &daoMock[bool]{resp: true},
			updateMock:  &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID2, 1)}},
			bytesMock:   withinBudget,
			updateBeats: []dao.BeatsUpdateItem{updateBeat(beatID2)},

			expect: []*core.Beat{coreBeat(beatID2, 1)},
		},
		{
			name: "Success/Batch",

			request: request(beat(beatID3), core.BeatsUpdateItem{
				ID: beatID1, Title: "Le thème", Color: "#E0457B", Description: "Paul ment.", Prompt: "Lent.",
			}),

			lockMock: &daoMock[bool]{resp: true},
			// The update returns its rows in no particular order.
			updateMock: &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID1, 0), daoBeat(beatID3, 2)}},
			bytesMock:  withinBudget,
			updateBeats: []dao.BeatsUpdateItem{
				updateBeat(beatID3),
				{ID: beatID1, Title: "Le thème", Color: "#e0457b", Description: "Paul ment.", Prompt: "Lent."},
			},

			expect: []*core.Beat{coreBeat(beatID3, 2), coreBeat(beatID1, 0)},
		},
		{
			name: "Error/InvalidRequest",

			request: request(core.BeatsUpdateItem{ID: beatID2, Color: "#4A90D9"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoBeat",

			request: request(),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyBeats",

			request: request(lo.Times(101, func(int) core.BeatsUpdateItem { return beat(uuid.New()) })...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateID",

			request: request(beat(beatID2), beat(beatID2)),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(beat(beatID2)),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(beat(beatID2)),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(beat(beatID2), beat(beatID3)),

			lockMock: &daoMock[bool]{resp: true},
			// One beat is missing, so the update writes none.
			updateMock:  &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID2, 1)}},
			updateBeats: []dao.BeatsUpdateItem{updateBeat(beatID2), updateBeat(beatID3)},

			expectErr: core.ErrBeatNotFound,
		},
		{
			name: "Error/Update",

			request: request(beat(beatID2)),

			lockMock:    &daoMock[bool]{resp: true},
			updateMock:  &daoMock[[]*dao.Beat]{err: errFoo},
			updateBeats: []dao.BeatsUpdateItem{updateBeat(beatID2)},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: request(beat(beatID2)),

			lockMock:    &daoMock[bool]{resp: true},
			updateMock:  &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID2, 1)}},
			bytesMock:   overBudget,
			updateBeats: []dao.BeatsUpdateItem{updateBeat(beatID2)},

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/ProjectBytes",

			request: request(beat(beatID2)),

			lockMock:    &daoMock[bool]{resp: true},
			updateMock:  &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID2, 1)}},
			bytesMock:   &daoMock[*dao.ProjectBytes]{err: errFoo},
			updateBeats: []dao.BeatsUpdateItem{updateBeat(beatID2)},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(beat(beatID2)),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockBeatsUpdateDaoLock(t)
			mockUpdate := coremocks.NewMockBeatsUpdateDao(t)
			mockBytes := coremocks.NewMockBeatsUpdateDaoBytes(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.BeatsUpdateRequest) bool {
						return req.ProjectID == projectID && recent(req.Now) && slices.Equal(req.Beats, testCase.updateBeats)
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

			service := core.NewBeatsUpdate(mockLock, mockUpdate, mockBytes, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
			mockBytes.AssertExpectations(t)
		})
	}
}
