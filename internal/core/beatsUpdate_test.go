package core_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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

		ownerMock  *daoMock[bool]
		updateMock *daoMock[[]*dao.Beat]
		// updateBeats are the beats the update receives.
		updateBeats []dao.BeatsUpdateItem

		expect    []*core.Beat
		expectErr error
	}{
		{
			name: "Success",

			request: request(beat(beatID2)),

			ownerMock:   &daoMock[bool]{resp: true},
			updateMock:  &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID2, 1)}},
			updateBeats: []dao.BeatsUpdateItem{updateBeat(beatID2)},

			expect: []*core.Beat{coreBeat(beatID2, 1)},
		},
		{
			name: "Success/Batch",

			request: request(beat(beatID3), core.BeatsUpdateItem{
				ID: beatID1, Title: "Le thème", Color: "#E0457B", Description: "Paul ment.", Prompt: "Lent.",
			}),

			ownerMock: &daoMock[bool]{resp: true},
			// The update returns its rows in no particular order.
			updateMock: &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID1, 0), daoBeat(beatID3, 2)}},
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

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request(beat(beatID2)),

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request(beat(beatID2), beat(beatID3)),

			ownerMock: &daoMock[bool]{resp: true},
			// One beat is missing, so the update writes none.
			updateMock:  &daoMock[[]*dao.Beat]{resp: []*dao.Beat{daoBeat(beatID2, 1)}},
			updateBeats: []dao.BeatsUpdateItem{updateBeat(beatID2), updateBeat(beatID3)},

			expectErr: core.ErrBeatNotFound,
		},
		{
			name: "Error/Update",

			request: request(beat(beatID2)),

			ownerMock:   &daoMock[bool]{resp: true},
			updateMock:  &daoMock[[]*dao.Beat]{err: errFoo},
			updateBeats: []dao.BeatsUpdateItem{updateBeat(beatID2)},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockBeatsUpdateDaoOwner(t)
			mockUpdate := coremocks.NewMockBeatsUpdateDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.BeatsUpdateRequest) bool {
						return req.ProjectID == projectID && recent(req.Now) && slices.Equal(req.Beats, testCase.updateBeats)
					})).
					Return(testCase.updateMock.resp, testCase.updateMock.err)
			}

			service := core.NewBeatsUpdate(mockOwner, mockUpdate)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
		})
	}
}
