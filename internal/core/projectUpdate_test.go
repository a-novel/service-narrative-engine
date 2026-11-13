package core_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestProjectUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.ProjectUpdateRequest{
		OwnerID: ownerID, ID: projectID, Title: "Clear Hours", Language: "en", Tags: []string{"noir", "huis-clos"},
		Genres: []string{"thriller"},
	}

	withTags := func(tags []string) *core.ProjectUpdateRequest {
		tagged := *request
		tagged.Tags = tags

		return &tagged
	}

	withGenres := func(genres []string) *core.ProjectUpdateRequest {
		classified := *request
		classified.Genres = genres

		return &classified
	}

	updated := &core.Project{
		ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr", Tags: []string{"noir"},
		Genres:    []string{"thriller"},
		CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt, LogoVersion: lo.ToPtr(editedAt),
		LogoChecksum: lo.ToPtr(logoChecksum),
	}

	testCases := []struct {
		name string

		request *core.ProjectUpdateRequest

		lockMock   *daoMock[bool]
		daoMock    *daoMock[*dao.Project]
		bytesMock  *daoMock[*dao.ProjectBytes]
		transactor transaction.Transactor

		expect    *core.Project
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock: withinBudget,

			expect: updated,
		},
		{
			name: "Success/ClearTags",

			request: withTags([]string{}),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock: withinBudget,

			expect: updated,
		},
		{
			name: "Success/ClearGenres",

			request: withGenres([]string{}),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock: withinBudget,

			expect: updated,
		},
		{
			name: "Success/Tags",

			// A tag takes lowercase letters from any script, with their marks, modifier letters and zero-width
			// joiners, and digits, and counts its runes.
			request: withTags([]string{
				"noir", "huis-clos", "années-50", "می\u200cخواهم", "क्\u200dष",
				"เรื่อง", "हिंदी", "ラーメン", "i\u0307stanbul", strings.Repeat("é", 24),
			}),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock: withinBudget,

			expect: updated,
		},
		{
			name: "Success/MostTags",

			request: withTags(tenLabels),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock: withinBudget,

			expect: updated,
		},
		{
			name: "Success/MostGenres",

			request: withGenres(tenLabels),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock: withinBudget,

			expect: updated,
		},
		{
			name: "Error/InvalidRequest",

			request: &core.ProjectUpdateRequest{
				OwnerID: ownerID, ID: projectID, Language: "en", Tags: []string{}, Genres: []string{},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/UnknownLanguage",

			request: &core.ProjectUpdateRequest{
				OwnerID: ownerID, ID: projectID, Title: "Clear Hours", Language: "de", Tags: []string{},
				Genres: []string{},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TitleTooLong",

			request: &core.ProjectUpdateRequest{
				OwnerID: ownerID, ID: projectID, Title: strings.Repeat("é", 81), Language: "en", Tags: []string{},
				Genres: []string{},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagsOmitted",

			request: withTags(nil),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/GenresOmitted",

			request: withGenres(nil),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyTags",

			request: withTags(append(slices.Clone(tenLabels), "k")),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyGenres",

			request: withGenres(append(slices.Clone(tenLabels), "k")),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateTags",

			request: withTags([]string{"noir", "noir"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateGenres",

			request: withGenres([]string{"noir", "noir"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/EmptyTag",

			request: withTags([]string{""}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/EmptyGenre",

			request: withGenres([]string{""}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/BlankTag",

			request: withTags([]string{"   "}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/BlankGenre",

			request: withGenres([]string{"   "}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagWhitespace",

			request: withTags([]string{"noir "}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/GenreWhitespace",

			request: withGenres([]string{"noir "}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagNUL",

			request: withTags([]string{"no\x00ir"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/GenreNUL",

			request: withGenres([]string{"no\x00ir"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagTooLong",

			request: withTags([]string{strings.Repeat("é", 25)}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/GenreTooLong",

			request: withGenres([]string{strings.Repeat("é", 25)}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagUppercase",

			request: withTags([]string{"Noir"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagSpace",

			request: withTags([]string{"huis clos"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagUnderscore",

			request: withTags([]string{"roman_maritime"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagLeadingHyphen",

			request: withTags([]string{"-noir"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagTrailingHyphen",

			request: withTags([]string{"noir-"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagDoubleHyphen",

			request: withTags([]string{"huis--clos"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagMarkOnly",

			// An emoji's variation selector is a mark that starts no word.
			request: withTags([]string{"\ufe0f"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagJoinerWord",

			// A zero-width joiner sits inside a word, never alone between two hyphens.
			request: withTags([]string{"rock-\u200d-star"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagDecomposedAccent",

			// The accent is a combining mark after the "e": the tag is not precomposed (NFC), where "années-50"
			// holds a precomposed "é".
			request: withTags([]string{"anne\u0301es-50"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request,

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/DeletedMeanwhile",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.Project]{err: dao.ErrProjectUpdateNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Internal",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: request,

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock: overBudget,

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/ProjectBytes",

			request: request,

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock: &daoMock[*dao.ProjectBytes]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request,

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockProjectUpdateDaoLock(t)
			mockDao := coremocks.NewMockProjectUpdateDao(t)
			mockBytes := coremocks.NewMockProjectUpdateDaoBytes(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectUpdateRequest) bool {
						return req.ID == projectID && req.Title == "Clear Hours" && req.Language == "en" && recent(req.Now) &&
							slices.Equal(req.Tags, testCase.request.Tags) && slices.Equal(req.Genres, testCase.request.Genres)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.bytesMock != nil {
				mockBytes.EXPECT().
					Exec(mock.Anything, &dao.ProjectBytesRequest{ID: projectID}).
					Return(testCase.bytesMock.resp, testCase.bytesMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewProjectUpdate(mockLock, mockDao, mockBytes, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockDao.AssertExpectations(t)
			mockBytes.AssertExpectations(t)
		})
	}
}
