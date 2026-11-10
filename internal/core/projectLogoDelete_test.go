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

func TestProjectLogoDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.ProjectLogoDeleteRequest{OwnerID: ownerID, ProjectID: projectID}

	testCases := []struct {
		name string

		request *core.ProjectLogoDeleteRequest

		ownerMock  *daoMock[bool]
		deleteMock *daoMock[*dao.Project]

		expect    *core.Project
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			deleteMock: &daoMock[*dao.Project]{resp: daoProject()},

			expect: coreProject(),
		},
		{
			name: "Error/MissingProjectID",

			request: &core.ProjectLogoDeleteRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Owner",

			request: request,

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Delete",

			request: request,

			ownerMock:  &daoMock[bool]{resp: true},
			deleteMock: &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockProjectLogoDeleteDaoOwner(t)
			mockDelete := coremocks.NewMockProjectLogoDeleteDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.ProjectLogoDeleteRequest{ProjectID: projectID}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
			}

			service := core.NewProjectLogoDelete(mockOwner, mockDelete)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
		})
	}
}
