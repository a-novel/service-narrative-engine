package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestIdeaVersionCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")
	validRequest := &core.IdeaVersionCreateRequest{
		Actor:  core.Actor{UserID: ownerID},
		IdeaID: ideaID,
		Seed:   "The answering foghorn moves closer.",
		Genre:  "speculative",
		Title:  "The Nearer Light",
	}
	versionID := uuid.MustParse("00000000-0000-0000-0000-000000000202")
	versionCreatedAt := createdAt.Add(time.Second)
	entity := &dao.IdeaVersion{
		ID:        versionID,
		IdeaID:    ideaID,
		Seed:      validRequest.Seed,
		Genre:     validRequest.Genre,
		Title:     validRequest.Title,
		CreatedAt: versionCreatedAt,
	}
	expect := &core.Idea{
		ID:        ideaID,
		OwnerID:   ownerID,
		Seed:      entity.Seed,
		Genre:     entity.Genre,
		Title:     entity.Title,
		CreatedAt: createdAt,
		UpdatedAt: &versionCreatedAt,
	}

	testCases := []struct {
		name string

		request *core.IdeaVersionCreateRequest

		accessErr  error
		callAccess bool
		daoResult  *dao.IdeaVersion
		daoErr     error
		callDao    bool

		transactorErr error
		expect        *core.Idea
		expectErr     error
	}{
		{
			name:       "Success",
			request:    validRequest,
			callAccess: true,
			daoResult:  entity,
			callDao:    true,
			expect:     expect,
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.IdeaVersionCreateRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Content",
			request: &core.IdeaVersionCreateRequest{
				Actor:  validRequest.Actor,
				IdeaID: ideaID,
				Seed:   " ",
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/AccessBeforeContentValidation",
			request: &core.IdeaVersionCreateRequest{
				Actor:  validRequest.Actor,
				IdeaID: ideaID,
				Seed:   " ",
			},
			accessErr:  core.ErrIdeaNotFound,
			callAccess: true,
			expectErr:  core.ErrIdeaNotFound,
		},
		{
			name:       "Error/OwnerRelock",
			request:    validRequest,
			callAccess: true,
			daoErr:     dao.ErrIdeaLockNotFound,
			callDao:    true,
			expectErr:  core.ErrIdeaNotFound,
		},
		{
			name:       "Error/Dao",
			request:    validRequest,
			callAccess: true,
			daoErr:     errFoo,
			callDao:    true,
			expectErr:  errFoo,
		},
		{
			name:          "Error/Transaction",
			request:       validRequest,
			callAccess:    true,
			transactorErr: errFoo,
			expectErr:     errFoo,
		},
		{
			name:       "Error/MissingEntity",
			request:    validRequest,
			callAccess: true,
			callDao:    true,
			expectErr:  errors.New("idea version insert returned no entity"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectAccess := coremocks.NewMockProjectAccessService(t)
			ideaVersionDao := coremocks.NewMockIdeaVersionInsertDao(t)

			if testCase.callAccess {
				projectAccess.EXPECT().
					Exec(mock.Anything, &core.ProjectAccessRequest{
						Actor:  testCase.request.Actor,
						IdeaID: testCase.request.IdeaID,
					}).
					Return(ideaFixture(), testCase.accessErr)
			}

			if testCase.callDao {
				ideaVersionDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.IdeaVersionInsertRequest) bool {
						return assert.NotEqual(t, uuid.Nil, request.ID) &&
							assert.Equal(t, testCase.request.IdeaID, request.IdeaID) &&
							assert.Equal(t, testCase.request.Actor.UserID, request.OwnerID) &&
							assert.Equal(t, testCase.request.Seed, request.Seed) &&
							assert.Equal(t, testCase.request.Genre, request.Genre) &&
							assert.Equal(t, testCase.request.Title, request.Title) &&
							assert.WithinDuration(t, time.Now(), request.Now, time.Minute)
					})).
					Return(testCase.daoResult, testCase.daoErr)
			}

			transactor := transactiontest.NewTransactor()
			if testCase.transactorErr != nil {
				transactor = transactiontest.NewFailingTransactor(testCase.transactorErr)
			}

			result, err := core.NewIdeaVersionCreate(
				projectAccess,
				ideaVersionDao,
				transactor,
			).Exec(t.Context(), testCase.request)
			if testCase.expectErr != nil && testCase.name == "Error/MissingEntity" {
				require.ErrorContains(t, err, testCase.expectErr.Error())
			} else {
				require.ErrorIs(t, err, testCase.expectErr)
			}

			require.Equal(t, testCase.expect, result)

			projectAccess.AssertExpectations(t)
			ideaVersionDao.AssertExpectations(t)
		})
	}
}
