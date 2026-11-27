package core_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/v2/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

func TestInstanceLinksList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	otherInstanceID := uuid.MustParse("00000000-0000-0000-0000-000000000032")

	links := json.RawMessage(`[{"source": "scenes/a"}]`)

	// The project holds three instances, and the page two of them: one wrote its links, the other never did.
	rows := []*dao.InstanceLinksListRow{
		{InstanceID: instanceID, Links: links, UpdatedAt: new(documentUpdatedAt), Total: 3},
		{InstanceID: otherInstanceID, Links: json.RawMessage(`[]`), Total: 3},
	}

	page := &core.InstanceLinksPage{
		Instances: []*core.InstanceLinks{
			{InstanceID: instanceID, Links: links, UpdatedAt: new(documentUpdatedAt)},
			{InstanceID: otherInstanceID, Links: json.RawMessage(`[]`)},
		},
		Total: 3,
	}

	testCases := []struct {
		name string

		request *core.InstanceLinksListRequest

		ownerMock *daoMock[bool]
		daoMock   *daoMock[[]*dao.InstanceLinksListRow]
		// expectRequest is the page the DAO is asked for.
		expectRequest *dao.InstanceLinksListRequest

		expect    *core.InstanceLinksPage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.InstanceLinksListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 2, Offset: 1},

			ownerMock:     &daoMock[bool]{resp: true},
			daoMock:       &daoMock[[]*dao.InstanceLinksListRow]{resp: rows},
			expectRequest: &dao.InstanceLinksListRequest{ProjectID: projectID, Limit: 2, Offset: 1},

			expect: page,
		},
		{
			// A request that names no limit takes the default page of the instance list.
			name: "Success/Defaults",

			request: &core.InstanceLinksListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock:     &daoMock[bool]{resp: true},
			daoMock:       &daoMock[[]*dao.InstanceLinksListRow]{resp: rows},
			expectRequest: &dao.InstanceLinksListRequest{ProjectID: projectID, Limit: core.InstanceListLimitDefault},

			expect: page,
		},
		{
			// A page past the end has no row to carry the total.
			name: "Success/Empty",

			request: &core.InstanceLinksListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 500, Offset: 3},

			ownerMock:     &daoMock[bool]{resp: true},
			daoMock:       &daoMock[[]*dao.InstanceLinksListRow]{resp: []*dao.InstanceLinksListRow{}},
			expectRequest: &dao.InstanceLinksListRequest{ProjectID: projectID, Limit: 500, Offset: 3},

			expect: &core.InstanceLinksPage{Instances: []*core.InstanceLinks{}},
		},
		{
			name: "Error/NoProject",

			request: &core.InstanceLinksListRequest{OwnerID: ownerID, ProjectID: uuid.Nil},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeLimit",

			request: &core.InstanceLinksListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/LimitTooLarge",

			request: &core.InstanceLinksListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 501},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeOffset",

			request: &core.InstanceLinksListRequest{OwnerID: ownerID, ProjectID: projectID, Offset: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: &core.InstanceLinksListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Internal",

			request: &core.InstanceLinksListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock:     &daoMock[bool]{resp: true},
			daoMock:       &daoMock[[]*dao.InstanceLinksListRow]{err: errFoo},
			expectRequest: &dao.InstanceLinksListRequest{ProjectID: projectID, Limit: core.InstanceListLimitDefault},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockProjectOwnerDao(t)
			mockDao := coremocks.NewMockInstanceLinksListDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, testCase.expectRequest).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewInstanceLinksList(mockOwner, mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
