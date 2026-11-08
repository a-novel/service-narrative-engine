package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestProjectPictureDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.ProjectPictureDeleteRequest{
		OwnerID: ownerID, ProjectID: projectID, Picture: core.ProjectPictureBackground,
	}

	testCases := []struct {
		name string

		request *core.ProjectPictureDeleteRequest

		lockMock   *daoMock[bool]
		deleteMock *daoMock[*dao.Project]
		transactor transaction.Transactor

		expect    *core.Project
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			lockMock:   &daoMock[bool]{resp: true},
			deleteMock: &daoMock[*dao.Project]{resp: daoProject()},

			expect: coreProject(),
		},
		{
			name: "Error/UnknownPicture",

			request: &core.ProjectPictureDeleteRequest{OwnerID: ownerID, ProjectID: projectID, Picture: "banner"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request,

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Delete",

			request: request,

			lockMock:   &daoMock[bool]{resp: true},
			deleteMock: &daoMock[*dao.Project]{err: errFoo},

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

			mockLock := coremocks.NewMockProjectPictureDeleteDaoLock(t)
			mockDelete := coremocks.NewMockProjectPictureDeleteDao(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.ProjectPictureDeleteRequest{
						ProjectID: projectID, Picture: core.ProjectPictureBackground,
					}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewProjectPictureDelete(mockLock, mockDelete, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
		})
	}
}
