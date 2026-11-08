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

func TestProjectPictureDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.ProjectPictureDeleteRequest{
		OwnerID: ownerID, ProjectID: projectID, Picture: core.ProjectPictureBackground,
	}

	testCases := []struct {
		name string

		request *core.ProjectPictureDeleteRequest

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
			name: "Error/UnknownPicture",

			request: &core.ProjectPictureDeleteRequest{OwnerID: ownerID, ProjectID: projectID, Picture: "banner"},

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

			mockOwner := coremocks.NewMockProjectPictureDeleteDaoOwner(t)
			mockDelete := coremocks.NewMockProjectPictureDeleteDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.deleteMock != nil {
				mockDelete.EXPECT().
					Exec(mock.Anything, &dao.ProjectPictureDeleteRequest{
						ProjectID: projectID, Picture: core.ProjectPictureBackground,
					}).
					Return(testCase.deleteMock.resp, testCase.deleteMock.err)
			}

			service := core.NewProjectPictureDelete(mockOwner, mockDelete)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockDelete.AssertExpectations(t)
		})
	}
}
