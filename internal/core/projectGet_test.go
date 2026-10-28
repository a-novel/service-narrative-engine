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

func TestProjectGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		request *core.ProjectGetRequest

		ownerMock *daoMock[bool]
		daoMock   *daoMock[*dao.Project]

		expect    *core.Project
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ProjectGetRequest{OwnerID: ownerID, ID: projectID},

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},

			expect: &core.Project{
				ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
				CreatedAt: createdAt, UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/MissingID",

			request: &core.ProjectGetRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: &core.ProjectGetRequest{OwnerID: ownerID, ID: projectID},

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: &core.ProjectGetRequest{OwnerID: ownerID, ID: projectID},

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Select",

			request: &core.ProjectGetRequest{OwnerID: ownerID, ID: projectID},

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockProjectGetDaoOwner(t)
			mockDao := coremocks.NewMockProjectGetDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewProjectGet(mockOwner, mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockDao.AssertExpectations(t)
		})
	}
}
