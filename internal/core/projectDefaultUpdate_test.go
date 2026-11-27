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

func TestProjectDefaultUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.ProjectDefaultUpdateRequest{OwnerID: ownerID, ProjectID: projectID, InstanceID: instanceID}

	stored := daoProject()
	stored.DefaultInstanceID = new(instanceID)

	chosen := coreProject()
	chosen.DefaultInstanceID = new(instanceID)

	testCases := []struct {
		name string

		request *core.ProjectDefaultUpdateRequest

		lockMock   *daoMock[bool]
		daoMock    *daoMock[*dao.Project]
		transactor transaction.Transactor

		expect    *core.Project
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.Project]{resp: stored},

			expect: chosen,
		},
		{
			name: "Error/NoInstance",

			request: &core.ProjectDefaultUpdateRequest{OwnerID: ownerID, ProjectID: projectID, InstanceID: uuid.Nil},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/InstanceNotFound",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.Project]{err: dao.ErrInstanceNotFound},

			expectErr: core.ErrInstanceNotFound,
		},
		{
			name: "Error/Internal",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.Project]{err: errFoo},

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
			mockDao := coremocks.NewMockProjectDefaultUpdateDao(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.daoMock != nil {
				// The choice stamps the project at an instant the service captures.
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectDefaultUpdateRequest) bool {
						return req.ProjectID == projectID && req.InstanceID == instanceID && recent(req.Now)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewProjectDefaultUpdate(mockLock, mockDao, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
