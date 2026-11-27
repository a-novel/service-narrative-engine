package core_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/v2/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

func TestInstanceDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.InstanceDeleteRequest{OwnerID: ownerID, ProjectID: projectID, ID: instanceID}

	testCases := []struct {
		name string

		request *core.InstanceDeleteRequest

		lockMock   *daoMock[bool]
		daoMock    *daoMock[*dao.InstanceDescriptor]
		transactor transaction.Transactor

		expect    *core.InstanceDescriptor
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},

			expect: coreDescriptor(),
		},
		{
			name: "Error/NoID",

			request: &core.InstanceDeleteRequest{OwnerID: ownerID, ProjectID: projectID, ID: uuid.Nil},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/NotFound",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.InstanceDescriptor]{err: dao.ErrInstanceNotFound},

			expectErr: core.ErrInstanceNotFound,
		},
		{
			name: "Error/Default",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.InstanceDescriptor]{err: dao.ErrInstanceDeleteDefault},

			expectErr: core.ErrInstanceDefault,
		},
		{
			name: "Error/Internal",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.InstanceDescriptor]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request,

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockProjectLockDao(t)
			mockDao := coremocks.NewMockInstanceDeleteDao(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.InstanceDeleteRequest{ProjectID: projectID, ID: instanceID}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewInstanceDelete(mockLock, mockDao, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
