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

func TestProjectAccess(t *testing.T) {
	t.Parallel()

	errDAO := errors.New("dao failure")
	validRequest := &core.ProjectAccessRequest{
		Actor:     core.Actor{UserID: ownerID},
		ProjectID: projectID,
	}

	testCases := []struct {
		name string

		request  *core.ProjectAccessRequest
		response *dao.Project
		daoErr   error
		callDAO  bool

		expect    *dao.Project
		expectErr error
	}{
		{
			name:     "Success",
			request:  validRequest,
			response: projectFixture(),
			callDAO:  true,
			expect:   projectFixture(),
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.ProjectAccessRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "Error/OtherOwnerOrAbsent",
			request:   validRequest,
			daoErr:    dao.ErrProjectSelectNotFound,
			callDAO:   true,
			expectErr: core.ErrProjectNotFound,
		},
		{
			name:      "Error/DAO",
			request:   validRequest,
			daoErr:    errDAO,
			callDAO:   true,
			expectErr: errDAO,
		},
		{
			name:      "Error/MissingEntity",
			request:   validRequest,
			callDAO:   true,
			expectErr: errors.New("project selection returned no entity"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDao := coremocks.NewMockProjectSelectDao(t)
			if testCase.callDAO {
				projectDao.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{
						ID:      testCase.request.ProjectID,
						OwnerID: testCase.request.Actor.UserID,
					}).
					Return(testCase.response, testCase.daoErr)
			}

			result, err := core.NewProjectAccess(projectDao).Exec(t.Context(), testCase.request)

			if testCase.expectErr != nil && !errors.Is(testCase.expectErr, core.ErrInvalidRequest) &&
				!errors.Is(testCase.expectErr, core.ErrProjectNotFound) &&
				!errors.Is(testCase.expectErr, errDAO) {
				require.ErrorContains(t, err, testCase.expectErr.Error())
			} else {
				require.ErrorIs(t, err, testCase.expectErr)
			}

			require.Equal(t, testCase.expect, result)
		})
	}
}
