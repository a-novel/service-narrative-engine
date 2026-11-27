package core_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/v2/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

func TestInstanceList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	// The project holds three instances, and the page the second of them.
	rows := []*dao.InstanceListRow{{InstanceDescriptor: *daoDescriptor(), Total: 3}}

	testCases := []struct {
		name string

		request *core.InstanceListRequest

		ownerMock *daoMock[bool]
		daoMock   *daoMock[[]*dao.InstanceListRow]
		// expectRequest is the page the DAO is asked for.
		expectRequest *dao.InstanceListRequest

		expect    *core.InstancePage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.InstanceListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 1, Offset: 1},

			ownerMock:     &daoMock[bool]{resp: true},
			daoMock:       &daoMock[[]*dao.InstanceListRow]{resp: rows},
			expectRequest: &dao.InstanceListRequest{ProjectID: projectID, Limit: 1, Offset: 1},

			expect: &core.InstancePage{Instances: []*core.InstanceDescriptor{coreDescriptor()}, Total: 3},
		},
		{
			// A request that names no limit takes the default page.
			name: "Success/Defaults",

			request: &core.InstanceListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock:     &daoMock[bool]{resp: true},
			daoMock:       &daoMock[[]*dao.InstanceListRow]{resp: rows},
			expectRequest: &dao.InstanceListRequest{ProjectID: projectID, Limit: core.InstanceListLimitDefault},

			expect: &core.InstancePage{Instances: []*core.InstanceDescriptor{coreDescriptor()}, Total: 3},
		},
		{
			// A page past the end has no row to carry the total.
			name: "Success/Empty",

			request: &core.InstanceListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 500, Offset: 3},

			ownerMock:     &daoMock[bool]{resp: true},
			daoMock:       &daoMock[[]*dao.InstanceListRow]{resp: []*dao.InstanceListRow{}},
			expectRequest: &dao.InstanceListRequest{ProjectID: projectID, Limit: 500, Offset: 3},

			expect: &core.InstancePage{Instances: []*core.InstanceDescriptor{}},
		},
		{
			name: "Error/NoProject",

			request: &core.InstanceListRequest{OwnerID: ownerID, ProjectID: uuid.Nil},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeLimit",

			request: &core.InstanceListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/LimitTooLarge",

			request: &core.InstanceListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 501},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeOffset",

			request: &core.InstanceListRequest{OwnerID: ownerID, ProjectID: projectID, Offset: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: &core.InstanceListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Internal",

			request: &core.InstanceListRequest{OwnerID: ownerID, ProjectID: projectID},

			ownerMock:     &daoMock[bool]{resp: true},
			daoMock:       &daoMock[[]*dao.InstanceListRow]{err: errFoo},
			expectRequest: &dao.InstanceListRequest{ProjectID: projectID, Limit: core.InstanceListLimitDefault},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockProjectOwnerDao(t)
			mockDao := coremocks.NewMockInstanceListDao(t)

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

			service := core.NewInstanceList(mockOwner, mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
