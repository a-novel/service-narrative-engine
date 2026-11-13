package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestProjectLogosDrift(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	removedID := uuid.MustParse("00000000-0000-0000-0000-0000000000ff")

	held := map[string]string{projectID.String(): logoChecksum}

	// full holds a logo past the most projects a user owns.
	full := make(map[string]string, core.ProjectsPerOwnerMax+1)
	for range core.ProjectsPerOwnerMax + 1 {
		full[uuid.NewString()] = logoChecksum
	}

	testCases := []struct {
		name string

		request *core.ProjectLogosDriftRequest

		daoRequest *dao.ProjectLogosDriftRequest
		daoMock    *daoMock[[]*dao.ProjectLogosDriftRow]

		expect    *core.ProjectLogosDriftResult
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ProjectLogosDriftRequest{OwnerID: ownerID, Logos: held},

			daoRequest: &dao.ProjectLogosDriftRequest{OwnerID: ownerID, Logos: held, Limit: 101},
			daoMock: &daoMock[[]*dao.ProjectLogosDriftRow]{
				resp: []*dao.ProjectLogosDriftRow{
					{ProjectID: projectID, Checksum: lo.ToPtr(logoChecksum)},
					{ProjectID: removedID},
				},
			},

			expect: &core.ProjectLogosDriftResult{
				Logos:   map[uuid.UUID]string{projectID: logoChecksum},
				Removed: []uuid.UUID{removedID},
			},
		},
		{
			name: "Success/NoDrift",

			request: &core.ProjectLogosDriftRequest{OwnerID: ownerID, Logos: map[string]string{}},

			daoRequest: &dao.ProjectLogosDriftRequest{OwnerID: ownerID, Logos: map[string]string{}, Limit: 100},
			daoMock:    &daoMock[[]*dao.ProjectLogosDriftRow]{},

			expect: &core.ProjectLogosDriftResult{Logos: map[uuid.UUID]string{}, Removed: []uuid.UUID{}},
		},
		{
			name: "Error/NoOwner",

			request: &core.ProjectLogosDriftRequest{Logos: held},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoLogos",

			request: &core.ProjectLogosDriftRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyLogos",

			request: &core.ProjectLogosDriftRequest{OwnerID: ownerID, Logos: full},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidID",

			request: &core.ProjectLogosDriftRequest{OwnerID: ownerID, Logos: map[string]string{"not-a-uuid": logoChecksum}},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/UppercaseChecksum",

			request: &core.ProjectLogosDriftRequest{
				OwnerID: ownerID, Logos: map[string]string{projectID.String(): strings.ToUpper(logoChecksum)},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ShortChecksum",

			request: &core.ProjectLogosDriftRequest{
				OwnerID: ownerID, Logos: map[string]string{projectID.String(): logoChecksum[:63]},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Internal",

			request: &core.ProjectLogosDriftRequest{OwnerID: ownerID, Logos: held},

			daoRequest: &dao.ProjectLogosDriftRequest{OwnerID: ownerID, Logos: held, Limit: 101},
			daoMock:    &daoMock[[]*dao.ProjectLogosDriftRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockProjectLogosDriftDao(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, testCase.daoRequest).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewProjectLogosDrift(mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockDao.AssertExpectations(t)
		})
	}
}
