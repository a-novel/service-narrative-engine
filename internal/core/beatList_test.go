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

func TestBeatList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		request *core.BeatListRequest

		ownerMock *daoMock[bool]
		// listRequest is the page the service asks for, when the case reaches the list.
		listRequest *dao.BeatListRequest
		listMock    *daoMock[[]*dao.BeatListRow]

		expect    *core.BeatPage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.BeatListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 50, Offset: 100},

			ownerMock:   &daoMock[bool]{resp: true},
			listRequest: &dao.BeatListRequest{ProjectID: projectID, Limit: 50, Offset: 100},
			listMock:    &daoMock[[]*dao.BeatListRow]{resp: []*dao.BeatListRow{{Beat: *daoBeat(beatID1, 100), Total: 120}}},

			expect: &core.BeatPage{Beats: []*core.Beat{coreBeat(beatID1, 100)}, Total: 120},
		},
		{
			name: "Success/PastEnd",

			request: &core.BeatListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock:   &daoMock[bool]{resp: true},
			listRequest: &dao.BeatListRequest{ProjectID: projectID, Limit: core.BeatListLimitDefault},
			listMock:    &daoMock[[]*dao.BeatListRow]{},

			expect: &core.BeatPage{Beats: []*core.Beat{}},
		},
		{
			name: "Error/LimitTooHigh",

			request: &core.BeatListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 501},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeLimit",

			request: &core.BeatListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeOffset",

			request: &core.BeatListRequest{OwnerID: ownerID, ProjectID: projectID, Offset: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: &core.BeatListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: &core.BeatListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/List",

			request: &core.BeatListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock:   &daoMock[bool]{resp: true},
			listRequest: &dao.BeatListRequest{ProjectID: projectID, Limit: core.BeatListLimitDefault},
			listMock:    &daoMock[[]*dao.BeatListRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockBeatListDaoOwner(t)
			mockList := coremocks.NewMockBeatListDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.listMock != nil {
				mockList.EXPECT().
					Exec(mock.Anything, testCase.listRequest).
					Return(testCase.listMock.resp, testCase.listMock.err)
			}

			service := core.NewBeatList(mockOwner, mockList)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockList.AssertExpectations(t)
		})
	}
}
