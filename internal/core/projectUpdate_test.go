package core_test

import (
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestProjectUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.ProjectUpdateRequest{OwnerID: ownerID, ID: projectID, Title: "Clear Hours", Language: "en"}

	testCases := []struct {
		name string

		request *core.ProjectUpdateRequest

		lockMock   *daoMock[bool]
		daoMock    *daoMock[*dao.Project]
		bytesMock  *daoMock[*dao.ProjectBytes]
		transactor transaction.Transactor

		expect    *core.Project
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock: withinBudget,

			expect: &core.Project{
				ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
				CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt, LogoVersion: lo.ToPtr(editedAt),
			},
		},
		{
			name: "Error/InvalidRequest",

			request: &core.ProjectUpdateRequest{OwnerID: ownerID, ID: projectID, Language: "en"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/UnknownLanguage",

			request: &core.ProjectUpdateRequest{OwnerID: ownerID, ID: projectID, Title: "Clear Hours", Language: "de"},

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
			name: "Error/DeletedMeanwhile",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.Project]{err: dao.ErrProjectUpdateNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Internal",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: request,

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock: overBudget,

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/ProjectBytes",

			request: request,

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock: &daoMock[*dao.ProjectBytes]{err: errFoo},

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

			mockLock := coremocks.NewMockProjectUpdateDaoLock(t)
			mockDao := coremocks.NewMockProjectUpdateDao(t)
			mockBytes := coremocks.NewMockProjectUpdateDaoBytes(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectUpdateRequest) bool {
						return req.ID == projectID && req.Title == "Clear Hours" && req.Language == "en" && recent(req.Now)
					})).
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

			service := core.NewProjectUpdate(mockLock, mockDao, mockBytes, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockDao.AssertExpectations(t)
			mockBytes.AssertExpectations(t)
		})
	}
}
