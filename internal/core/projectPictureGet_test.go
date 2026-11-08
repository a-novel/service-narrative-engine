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

func TestProjectPictureGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.ProjectPictureGetRequest{OwnerID: ownerID, ProjectID: projectID, Picture: core.ProjectPictureLogo}

	testCases := []struct {
		name string

		request *core.ProjectPictureGetRequest

		ownerMock *daoMock[bool]
		daoMock   *daoMock[*dao.ProjectPicture]

		expect    *core.ProjectPicture
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock: &daoMock[*dao.ProjectPicture]{resp: &dao.ProjectPicture{
				ProjectID: projectID, Picture: core.ProjectPictureLogo, Data: []byte("logo"), MediaType: "image/webp",
				UpdatedAt: editedAt,
			}},

			expect: &core.ProjectPicture{MediaType: "image/webp", Data: []byte("logo"), Version: editedAt},
		},
		{
			name: "Error/UnknownPicture",

			request: &core.ProjectPictureGetRequest{OwnerID: ownerID, ProjectID: projectID, Picture: "banner"},

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
			name: "Error/NoPicture",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.ProjectPicture]{err: dao.ErrProjectPictureSelectNotFound},

			expectErr: core.ErrPictureNotFound,
		},
		{
			name: "Error/Select",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.ProjectPicture]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockProjectPictureGetDaoOwner(t)
			mockSelect := coremocks.NewMockProjectPictureGetDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.daoMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.ProjectPictureSelectRequest{ProjectID: projectID, Picture: core.ProjectPictureLogo}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewProjectPictureGet(mockOwner, mockSelect)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockSelect.AssertExpectations(t)
		})
	}
}
