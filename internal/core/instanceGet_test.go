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

func TestInstanceGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.InstanceGetRequest{OwnerID: ownerID, ProjectID: projectID, ID: instanceID}

	stored := &dao.Instance{
		InstanceDescriptor: *daoDescriptor(),
		Content:            json.RawMessage(`{"beats": []}`),
	}

	testCases := []struct {
		name string

		request *core.InstanceGetRequest

		ownerMock *daoMock[bool]
		daoMock   *daoMock[*dao.Instance]

		expect    *core.Instance
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Instance]{resp: stored},

			expect: &core.Instance{
				InstanceDescriptor: *coreDescriptor(),
				Content:            json.RawMessage(`{"beats": []}`),
			},
		},
		{
			name: "Error/NoID",

			request: &core.InstanceGetRequest{OwnerID: ownerID, ProjectID: projectID, ID: uuid.Nil},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/NotFound",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Instance]{err: dao.ErrInstanceNotFound},

			expectErr: core.ErrInstanceNotFound,
		},
		{
			name: "Error/Internal",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Instance]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockProjectOwnerDao(t)
			mockDao := coremocks.NewMockInstanceGetDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.InstanceSelectRequest{ProjectID: projectID, ID: instanceID}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewInstanceGet(mockOwner, mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
