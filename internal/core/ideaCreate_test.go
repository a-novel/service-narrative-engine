package core_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestIdeaCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")
	validRequest := &core.IdeaCreateRequest{
		Actor: core.Actor{UserID: ownerID},
		Seed:  "A lighthouse keeper hears a second foghorn.",
		Genre: "speculative",
		Title: "The Answering Light",
	}
	entity := ideaFixture()
	entityWithoutTitle := ideaFixture()
	entityWithoutTitle.Title = ""
	emptyEntity := ideaFixture()
	emptyEntity.Seed = ""
	emptyEntity.Genre = ""
	emptyEntity.Title = ""
	titleJustBelowLimitEntity := ideaFixture()
	titleJustBelowLimitEntity.Title = strings.Repeat("é", 127)
	titleAtLimitEntity := ideaFixture()
	titleAtLimitEntity.Title = strings.Repeat("é", 128)
	seedJustBelowLimitEntity := ideaFixture()
	seedJustBelowLimitEntity.Seed = strings.Repeat("é", 32767)
	seedAtLimitEntity := ideaFixture()
	seedAtLimitEntity.Seed = strings.Repeat("é", 32768)

	testCases := []struct {
		name string

		request   *core.IdeaCreateRequest
		daoResult *dao.Idea
		daoErr    error
		callDao   bool

		expect    *core.Idea
		expectErr error
	}{
		{
			name:    "Success",
			request: validRequest,
			callDao: true,
			expect: &core.Idea{
				ID:        entity.ID,
				OwnerID:   entity.OwnerID,
				Seed:      entity.Seed,
				Genre:     entity.Genre,
				Title:     entity.Title,
				CreatedAt: entity.CreatedAt,
			},
		},
		{
			name: "Success/WithoutTitle",
			request: &core.IdeaCreateRequest{
				Actor: core.Actor{UserID: ownerID},
				Seed:  validRequest.Seed,
				Genre: validRequest.Genre,
			},
			callDao:   true,
			daoResult: entityWithoutTitle,
			expect: &core.Idea{
				ID:        entityWithoutTitle.ID,
				OwnerID:   entityWithoutTitle.OwnerID,
				Seed:      entityWithoutTitle.Seed,
				Genre:     entityWithoutTitle.Genre,
				CreatedAt: entityWithoutTitle.CreatedAt,
			},
		},
		{
			name: "Success/EmptyPartialIdea",
			request: &core.IdeaCreateRequest{
				Actor: validRequest.Actor,
			},
			daoResult: emptyEntity,
			callDao:   true,
			expect: &core.Idea{
				ID:        emptyEntity.ID,
				OwnerID:   emptyEntity.OwnerID,
				CreatedAt: emptyEntity.CreatedAt,
			},
		},
		{
			name: "Success/TitleJustBelowLimit",
			request: &core.IdeaCreateRequest{
				Actor: validRequest.Actor,
				Seed:  validRequest.Seed,
				Genre: validRequest.Genre,
				Title: titleJustBelowLimitEntity.Title,
			},
			daoResult: titleJustBelowLimitEntity,
			callDao:   true,
			expect:    ideaFromEntity(titleJustBelowLimitEntity),
		},
		{
			name: "Success/TitleAtLimit",
			request: &core.IdeaCreateRequest{
				Actor: validRequest.Actor,
				Seed:  validRequest.Seed,
				Genre: validRequest.Genre,
				Title: titleAtLimitEntity.Title,
			},
			daoResult: titleAtLimitEntity,
			callDao:   true,
			expect:    ideaFromEntity(titleAtLimitEntity),
		},
		{
			name: "Success/SeedJustBelowLimit",
			request: &core.IdeaCreateRequest{
				Actor: validRequest.Actor,
				Seed:  seedJustBelowLimitEntity.Seed,
				Genre: validRequest.Genre,
			},
			daoResult: seedJustBelowLimitEntity,
			callDao:   true,
			expect:    ideaFromEntity(seedJustBelowLimitEntity),
		},
		{
			name: "Success/SeedAtLimit",
			request: &core.IdeaCreateRequest{
				Actor: validRequest.Actor,
				Seed:  seedAtLimitEntity.Seed,
				Genre: validRequest.Genre,
			},
			daoResult: seedAtLimitEntity,
			callDao:   true,
			expect:    ideaFromEntity(seedAtLimitEntity),
		},
		{
			name:      "Error/Actor",
			request:   &core.IdeaCreateRequest{Seed: validRequest.Seed, Genre: validRequest.Genre},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/SeedBlank",
			request: &core.IdeaCreateRequest{
				Actor: validRequest.Actor,
				Seed:  " \t ",
				Genre: validRequest.Genre,
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/SeedTooLong",
			request: &core.IdeaCreateRequest{
				Actor: validRequest.Actor,
				Seed:  strings.Repeat("é", 32769),
				Genre: validRequest.Genre,
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/GenreBlank",
			request: &core.IdeaCreateRequest{
				Actor: validRequest.Actor,
				Seed:  validRequest.Seed,
				Genre: " ",
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/GenreTooLong",
			request: &core.IdeaCreateRequest{
				Actor: validRequest.Actor,
				Seed:  validRequest.Seed,
				Genre: strings.Repeat("g", 129),
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TitleTooLong",
			request: &core.IdeaCreateRequest{
				Actor: validRequest.Actor,
				Seed:  validRequest.Seed,
				Genre: validRequest.Genre,
				Title: strings.Repeat("t", 129),
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "Error/Dao",
			request:   validRequest,
			daoErr:    errFoo,
			callDao:   true,
			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ideaDao := coremocks.NewMockIdeaCreateDao(t)

			if testCase.callDao {
				daoResult := testCase.daoResult
				if daoResult == nil {
					daoResult = entity
				}

				ideaDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.IdeaInsertRequest) bool {
						return assert.NotEqual(t, uuid.Nil, request.ID) &&
							assert.Equal(t, ownerID, request.OwnerID) &&
							assert.Equal(t, testCase.request.Seed, request.Seed) &&
							assert.Equal(t, testCase.request.Genre, request.Genre) &&
							assert.Equal(t, testCase.request.Title, request.Title) &&
							assert.WithinDuration(t, time.Now(), request.Now, time.Minute)
					})).
					Return(daoResult, testCase.daoErr)
			}

			result, err := core.NewIdeaCreate(ideaDao).Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}

func ideaFromEntity(entity *dao.Idea) *core.Idea {
	return &core.Idea{
		ID:        entity.ID,
		OwnerID:   entity.OwnerID,
		Seed:      entity.Seed,
		Genre:     entity.Genre,
		Title:     entity.Title,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}
