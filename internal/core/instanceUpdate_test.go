package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/v2/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

func TestInstanceUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	named := func(name string) *core.InstanceUpdateRequest {
		return &core.InstanceUpdateRequest{OwnerID: ownerID, ProjectID: projectID, ID: instanceID, Name: name}
	}

	testCases := []struct {
		name string

		request *core.InstanceUpdateRequest

		lockMock   *daoMock[bool]
		daoMock    *daoMock[*dao.InstanceDescriptor]
		bytesMock  *daoMock[*dao.ProjectBytes]
		transactor transaction.Transactor

		expect    *core.InstanceDescriptor
		expectErr error
	}{
		{
			name: "Success",

			request: named("Outline"),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			name: "Success/LongestName",

			request: named(strings.Repeat("é", core.InstanceNameMax)),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			// An empty name clears the one the instance held.
			name: "Success/NoName",

			request: named(""),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			name: "Error/BlankName",

			request: named(" "),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NameTooLong",

			request: named(strings.Repeat("é", core.InstanceNameMax+1)),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: named("Outline"),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/NotFound",

			request: named("Outline"),

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.InstanceDescriptor]{err: dao.ErrInstanceNotFound},

			expectErr: core.ErrInstanceNotFound,
		},
		{
			name: "Error/Internal",

			request: named("Outline"),

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.InstanceDescriptor]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: named("Outline"),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: overBudget,

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/Transaction",

			request: named("Outline"),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockProjectLockDao(t)
			mockDao := coremocks.NewMockInstanceUpdateDao(t)
			mockBytes := coremocks.NewMockProjectBytesDao(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.InstanceUpdateRequest{
						ProjectID: projectID, ID: instanceID, Name: testCase.request.Name,
					}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
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

			service := core.NewInstanceUpdate(mockLock, mockDao, mockBytes, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
