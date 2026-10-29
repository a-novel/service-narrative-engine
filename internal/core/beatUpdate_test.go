package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestBeatUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.BeatUpdateRequest{
		OwnerID: ownerID, ProjectID: projectID, ID: beatID2, Title: "Beat", Color: "#4A90D9",
	}

	testCases := []struct {
		name string

		request *core.BeatUpdateRequest

		ownerMock  *daoMock[bool]
		updateMock *daoMock[*dao.Beat]

		expect    *core.Beat
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Beat]{resp: daoBeat(beatID2, 1)},

			expect: coreBeat(beatID2, 1),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.BeatUpdateRequest{OwnerID: ownerID, ProjectID: projectID, ID: beatID2, Color: "#4A90D9"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request,

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Beat]{err: dao.ErrBeatUpdateNotFound},

			expectErr: core.ErrBeatNotFound,
		},
		{
			name: "Error/Update",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			updateMock: &daoMock[*dao.Beat]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockBeatUpdateDaoOwner(t)
			mockUpdate := coremocks.NewMockBeatUpdateDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.updateMock != nil {
				mockUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.BeatUpdateRequest) bool {
						return req.ProjectID == projectID && req.ID == beatID2 &&
							req.Title == "Beat" && req.Color == beatColor && recent(req.Now)
					})).
					Return(testCase.updateMock.resp, testCase.updateMock.err)
			}

			service := core.NewBeatUpdate(mockOwner, mockUpdate)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockUpdate.AssertExpectations(t)
		})
	}
}
