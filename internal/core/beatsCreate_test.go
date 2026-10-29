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

func TestBeatsCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	beat := core.BeatsCreateItem{Title: "Beat", Color: "#4A90D9"}
	// insertBeat is beat as the insert receives it, without its generated ID.
	insertBeat := dao.BeatsInsertItem{Title: "Beat", Color: beatColor}

	request := func(position int, beats ...core.BeatsCreateItem) *core.BeatsCreateRequest {
		return &core.BeatsCreateRequest{OwnerID: ownerID, ProjectID: projectID, Position: position, Beats: beats}
	}

	testCases := []struct {
		name string

		request *core.BeatsCreateRequest

		lockMock   *daoMock[bool]
		insertMock *daoMock[[]*dao.Beat]
		// insertBeats are the beats the insert receives, without their generated IDs.
		insertBeats []dao.BeatsInsertItem
		transactor  transaction.Transactor

		expect    []*core.Beat
		expectErr error
	}{
		{
			name: "Success/First",

			request: request(0, beat),

			lockMock:    &daoMock[bool]{resp: true},
			insertMock:  &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID3, 0)}},
			insertBeats: []dao.BeatsInsertItem{insertBeat},

			expect: []*core.Beat{coreBeat(beatID3, 0)},
		},
		{
			name: "Success/Position",

			request: request(2, beat),

			lockMock:    &daoMock[bool]{resp: true},
			insertMock:  &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID3, 2)}},
			insertBeats: []dao.BeatsInsertItem{insertBeat},

			expect: []*core.Beat{coreBeat(beatID3, 2)},
		},
		{
			name: "Success/Batch",

			request: request(1, beat, core.BeatsCreateItem{
				Title: "Le thème", Color: "#E0457B", Description: "Paul ment.", Prompt: "Lent.",
			}),

			lockMock: &daoMock[bool]{resp: true},
			// The insert returns its rows in no particular order.
			insertMock: &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID2, 2), daoBeat(beatID1, 1)}},
			insertBeats: []dao.BeatsInsertItem{
				insertBeat,
				{Title: "Le thème", Color: "#e0457b", Description: "Paul ment.", Prompt: "Lent."},
			},

			expect: []*core.Beat{coreBeat(beatID1, 1), coreBeat(beatID2, 2)},
		},
		{
			name: "Error/InvalidRequest",

			request: request(0, core.BeatsCreateItem{Color: "#4A90D9"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativePosition",

			request: request(-1, beat),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoBeat",

			request: request(0),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyBeats",

			request: request(0, slices.Repeat([]core.BeatsCreateItem{beat}, 101)...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(0, beat),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(0, beat),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectFull",

			request: request(0, beat),

			lockMock:    &daoMock[bool]{resp: true},
			insertMock:  &daoMock[[]*dao.Beat]{err: dao.ErrBeatsInsertFull},
			insertBeats: []dao.BeatsInsertItem{insertBeat},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/Insert",

			request: request(0, beat),

			lockMock:    &daoMock[bool]{resp: true},
			insertMock:  &daoMock[[]*dao.Beat]{err: errFoo},
			insertBeats: []dao.BeatsInsertItem{insertBeat},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(0, beat),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockBeatsCreateDaoLock(t)
			mockInsert := coremocks.NewMockBeatsCreateDaoInsert(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.insertMock != nil {
				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.BeatsInsertRequest) bool {
						ids := lo.Map(req.Beats, func(item dao.BeatsInsertItem, _ int) uuid.UUID { return item.ID })
						beats := lo.Map(req.Beats, func(item dao.BeatsInsertItem, _ int) dao.BeatsInsertItem {
							item.ID = uuid.Nil

							return item
						})

						return req.ProjectID == projectID && req.Position == testCase.request.Position && recent(req.Now) &&
							freshIDs(ids) && slices.Equal(beats, testCase.insertBeats)
					})).
					Return(testCase.insertMock.resp, testCase.insertMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewBeatsCreate(mockLock, mockInsert, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
		})
	}
}
