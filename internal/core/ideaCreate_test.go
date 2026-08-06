package core_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestIdeaCreate(t *testing.T) {
	t.Parallel()

	errDAO := errors.New("dao failure")
	validRequest := &core.IdeaCreateRequest{
		Actor: core.Actor{UserID: ownerID},
		Title: "The Answering Light",
		Genre: "speculative",
		Seed:  "A foghorn answers from beneath the sea.",
	}

	testCases := []struct {
		name string

		request   *core.IdeaCreateRequest
		callDAO   bool
		daoErr    error
		nilEntity bool

		expectErr error
	}{
		{
			name: "Success/StaticContractAtLimits",
			request: &core.IdeaCreateRequest{
				Actor: core.Actor{UserID: ownerID},
				Title: strings.Repeat("界", 128),
				Genre: "science-fiction",
				Seed:  strings.Repeat("é", 32_768),
			},
			callDAO: true,
		},
		{
			name:      "Error/IncompleteIdea",
			request:   &core.IdeaCreateRequest{Actor: core.Actor{UserID: ownerID}},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "Error/Anonymous",
			request:   &core.IdeaCreateRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TitleOverLimit",
			request: &core.IdeaCreateRequest{
				Actor: core.Actor{UserID: ownerID},
				Title: strings.Repeat("t", 129),
				Genre: "speculative",
				Seed:  "A foghorn answers from beneath the sea.",
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/BlankSeed",
			request: &core.IdeaCreateRequest{
				Actor: core.Actor{UserID: ownerID},
				Title: "The Answering Light",
				Genre: "speculative",
				Seed:  "   ",
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "Error/DAO",
			request:   validRequest,
			callDAO:   true,
			daoErr:    errDAO,
			expectErr: errDAO,
		},
		{
			name:      "Error/MissingEntity",
			request:   validRequest,
			callDAO:   true,
			nilEntity: true,
			expectErr: errors.New("idea insert returned no entity"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ideaDao := coremocks.NewMockIdeaCreateDao(t)
			if testCase.callDAO {
				ideaDao.EXPECT().
					Exec(mock.Anything, mock.Anything).
					RunAndReturn(func(
						_ context.Context,
						request *dao.IdeaInsertRequest,
					) (*dao.Idea, error) {
						if testCase.daoErr != nil || testCase.nilEntity {
							return nil, testCase.daoErr
						}

						require.NotEqual(t, request.ProjectID, request.VersionID)
						require.Equal(t, ownerID, request.OwnerID)
						require.Equal(t, testCase.request.Seed, request.Seed)
						require.Equal(t, testCase.request.Genre, request.Genre)
						require.Equal(t, testCase.request.Title, request.Title)

						return &dao.Idea{
							ProjectID:        request.ProjectID,
							VersionID:        request.VersionID,
							OwnerID:          request.OwnerID,
							Seed:             request.Seed,
							Genre:            request.Genre,
							Title:            request.Title,
							ProjectCreatedAt: request.Now,
							CreatedAt:        request.Now,
						}, nil
					})
			}

			result, err := core.NewIdeaCreate(ideaDao).Exec(t.Context(), testCase.request)

			if testCase.expectErr != nil &&
				!errors.Is(testCase.expectErr, core.ErrInvalidRequest) &&
				!errors.Is(testCase.expectErr, errDAO) {
				require.ErrorContains(t, err, testCase.expectErr.Error())
			} else {
				require.ErrorIs(t, err, testCase.expectErr)
			}

			if testCase.expectErr == nil {
				require.Equal(t, testCase.request.Seed, result.Seed)
				require.Equal(t, testCase.request.Genre, result.Genre)
				require.Equal(t, testCase.request.Title, result.Title)
				require.Equal(t, ownerID, result.OwnerID)
				require.NotEqual(t, result.ProjectID, result.VersionID)
			}
		})
	}
}
