package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestIdeaVersionCreate(t *testing.T) {
	t.Parallel()

	errAccess := errors.New("access failure")
	errDAO := errors.New("dao failure")
	errTx := errors.New("transaction failure")
	validRequest := &core.IdeaVersionCreateRequest{
		Actor:     core.Actor{UserID: ownerID},
		ProjectID: projectID,
		Seed:      "A foghorn answers from beneath the sea.",
		Genre:     "speculative",
		Title:     "The Answering Light",
	}

	testCases := []struct {
		name string

		request     *core.IdeaVersionCreateRequest
		accessErr   error
		daoErr      error
		transactErr error
		callAccess  bool
		callDAO     bool
		nilEntity   bool

		expectErr error
	}{
		{
			name:       "Success",
			request:    validRequest,
			callAccess: true,
			callDAO:    true,
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.IdeaVersionCreateRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectAccessBeforeContentValidation",
			request: &core.IdeaVersionCreateRequest{
				Actor:     core.Actor{UserID: ownerID},
				ProjectID: projectID,
				Seed:      "   ",
			},
			callAccess: true,
			accessErr:  errAccess,
			expectErr:  errAccess,
		},
		{
			name: "Error/StaticContract",
			request: &core.IdeaVersionCreateRequest{
				Actor:     core.Actor{UserID: ownerID},
				ProjectID: projectID,
				Seed:      "   ",
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name:       "Error/OwnerChangedBeforeWrite",
			request:    validRequest,
			callAccess: true,
			callDAO:    true,
			daoErr:     dao.ErrProjectLockNotFound,
			expectErr:  core.ErrProjectNotFound,
		},
		{
			name:        "Error/Transaction",
			request:     validRequest,
			callAccess:  true,
			transactErr: errTx,
			expectErr:   errTx,
		},
		{
			name:       "Error/MissingEntity",
			request:    validRequest,
			callAccess: true,
			callDAO:    true,
			nilEntity:  true,
			expectErr:  errors.New("idea version insert returned no entity"),
		},
		{
			name:       "Error/DAO",
			request:    validRequest,
			callAccess: true,
			callDAO:    true,
			daoErr:     errDAO,
			expectErr:  errDAO,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectAccess := coremocks.NewMockProjectAccessService(t)
			ideaDao := coremocks.NewMockIdeaVersionInsertDao(t)

			if testCase.callAccess {
				projectAccess.EXPECT().
					Exec(mock.Anything, &core.ProjectAccessRequest{
						Actor:     testCase.request.Actor,
						ProjectID: testCase.request.ProjectID,
					}).
					Return(projectFixture(), testCase.accessErr)
			}

			if testCase.callDAO {
				ideaDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.IdeaVersionInsertRequest) bool {
						return request.ProjectID == testCase.request.ProjectID &&
							request.OwnerID == ownerID &&
							request.Seed == testCase.request.Seed &&
							request.Genre == testCase.request.Genre &&
							request.Title == testCase.request.Title
					})).
					RunAndReturn(func(
						_ context.Context,
						request *dao.IdeaVersionInsertRequest,
					) (*dao.IdeaVersion, error) {
						if testCase.daoErr != nil || testCase.nilEntity {
							return nil, testCase.daoErr
						}

						return &dao.IdeaVersion{
							ID:        request.ID,
							ProjectID: request.ProjectID,
							Seed:      request.Seed,
							Genre:     request.Genre,
							Title:     request.Title,
							CreatedAt: request.Now,
						}, nil
					})
			}

			transactor := transactiontest.NewTransactor()
			if testCase.transactErr != nil {
				transactor = transactiontest.NewFailingTransactor(testCase.transactErr)
			}

			result, err := core.NewIdeaVersionCreate(projectAccess, ideaDao, transactor).
				Exec(t.Context(), testCase.request)

			if testCase.expectErr == nil {
				require.NoError(t, err)
			} else if errors.Is(err, testCase.expectErr) {
				require.ErrorIs(t, err, testCase.expectErr)
			} else {
				require.ErrorContains(t, err, testCase.expectErr.Error())
			}

			if testCase.expectErr == nil {
				require.Equal(t, projectID, result.ProjectID)
				require.Equal(t, ownerID, result.OwnerID)
				require.Equal(t, validRequest.Seed, result.Seed)
				require.NotEqual(t, projectID, result.VersionID)
			}
		})
	}
}
