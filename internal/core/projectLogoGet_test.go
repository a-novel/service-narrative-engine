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

func TestProjectLogoGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.ProjectLogoGetRequest{OwnerID: ownerID, ProjectID: projectID}

	testCases := []struct {
		name string

		request *core.ProjectLogoGetRequest

		ownerMock *daoMock[bool]
		daoMock   *daoMock[*dao.ProjectLogo]

		expect    *core.ProjectLogo
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock: &daoMock[*dao.ProjectLogo]{resp: &dao.ProjectLogo{
				ProjectID: projectID, Data: []byte("logo"), MediaType: "image/webp", Checksum: logoChecksum,
				UpdatedAt: editedAt,
			}},

			expect: &core.ProjectLogo{
				MediaType: "image/webp", Data: []byte("logo"), Checksum: logoChecksum, UpdatedAt: editedAt,
			},
		},
		{
			name: "Error/MissingProjectID",

			request: &core.ProjectLogoGetRequest{OwnerID: ownerID},

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
			name: "Error/NoLogo",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.ProjectLogo]{err: dao.ErrProjectLogoSelectNotFound},

			expectErr: core.ErrProjectLogoNotFound,
		},
		{
			name: "Error/Select",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.ProjectLogo]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockProjectLogoGetDaoOwner(t)
			mockSelect := coremocks.NewMockProjectLogoGetDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.daoMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.ProjectLogoSelectRequest{ProjectID: projectID}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewProjectLogoGet(mockOwner, mockSelect)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockSelect.AssertExpectations(t)
		})
	}
}
