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

	errFoo := errors.New("foo")
	validRequest := &core.ProjectAccessRequest{
		Actor:  core.Actor{UserID: ownerID},
		IdeaID: ideaID,
	}

	testCases := []struct {
		name string

		request *core.ProjectAccessRequest

		daoResult *dao.Idea
		daoErr    error
		callDao   bool

		expect    *dao.Idea
		expectErr error
	}{
		{
			name:      "Owner",
			request:   validRequest,
			daoResult: ideaFixture(),
			callDao:   true,
			expect:    ideaFixture(),
		},
		{
			name:      "InvalidRequest",
			request:   &core.ProjectAccessRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "HiddenOtherOwnerOrAbsent",
			request:   validRequest,
			daoErr:    dao.ErrIdeaSelectNotFound,
			callDao:   true,
			expectErr: core.ErrIdeaNotFound,
		},
		{
			name:      "Dao",
			request:   validRequest,
			daoErr:    errFoo,
			callDao:   true,
			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ideaDao := coremocks.NewMockIdeaSelectDao(t)
			if testCase.callDao {
				ideaDao.EXPECT().
					Exec(mock.Anything, &dao.IdeaSelectRequest{
						ID:      testCase.request.IdeaID,
						OwnerID: testCase.request.Actor.UserID,
					}).
					Return(testCase.daoResult, testCase.daoErr)
			}

			result, err := core.NewProjectAccess(ideaDao).Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
