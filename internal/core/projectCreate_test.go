package core_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestProjectCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	manuscript := daoLayer(layerID1, 0, core.LayerKindManuscript)
	manuscript.Name = "Manuscript"

	request := func(title, language string) *core.ProjectCreateRequest {
		return &core.ProjectCreateRequest{OwnerID: ownerID, Title: title, Premise: "Paul ment.", Language: language}
	}

	withTags := func(tags ...string) *core.ProjectCreateRequest {
		tagged := request("Les Heures claires", "fr")
		tagged.Tags = tags

		return tagged
	}

	withGenres := func(genres ...string) *core.ProjectCreateRequest {
		classified := request("Les Heures claires", "fr")
		classified.Genres = genres

		return classified
	}

	created := &core.ProjectComposer{
		Project: &core.Project{
			ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr", Tags: []string{"noir"},
			Genres:    []string{"thriller"},
			CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt, LogoVersion: lo.ToPtr(editedAt),
		},
		Layers: []*core.Layer{
			{
				ID: layerID1, ProjectID: projectID, Kind: core.LayerKindManuscript, Name: "Manuscript",
				UpdatedAt: updatedAt,
			},
		},
	}

	testCases := []struct {
		name string

		request    *core.ProjectCreateRequest
		transactor transaction.Transactor

		lockMock    *daoMock[struct{}]
		countMock   *daoMock[*dao.ProjectCount]
		projectMock *daoMock[*dao.Project]
		layersMock  *daoMock[[]*dao.Layer]

		expect    *core.ProjectComposer
		expectErr error
	}{
		{
			name: "Success",

			request: request("Les Heures claires", "fr"),

			lockMock:    &daoMock[struct{}]{},
			countMock:   &daoMock[*dao.ProjectCount]{resp: &dao.ProjectCount{Count: core.ProjectsPerOwnerMax - 1}},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layersMock:  &daoMock[[]*dao.Layer]{resp: []*dao.Layer{manuscript}},

			expect: created,
		},
		{
			name: "Success/Tags",

			// A tag takes lowercase letters from any script, with their marks, modifier letters and zero-width
			// joiners, and digits, and counts its runes.
			request: withTags(
				"noir", "huis-clos", "années-50", "می\u200cخواهم", "क्\u200dष",
				"เรื่อง", "हिंदी", "ラーメン", "i\u0307stanbul", strings.Repeat("é", 24),
			),

			lockMock:    &daoMock[struct{}]{},
			countMock:   &daoMock[*dao.ProjectCount]{resp: &dao.ProjectCount{Count: core.ProjectsPerOwnerMax - 1}},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layersMock:  &daoMock[[]*dao.Layer]{resp: []*dao.Layer{manuscript}},

			expect: created,
		},
		{
			name: "Success/Genres",

			// The check is exact, so a change of case is another genre, and a genre counts its runes.
			request: withGenres("noir", "Noir", "huis clos", strings.Repeat("é", 24)),

			lockMock:    &daoMock[struct{}]{},
			countMock:   &daoMock[*dao.ProjectCount]{resp: &dao.ProjectCount{Count: core.ProjectsPerOwnerMax - 1}},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layersMock:  &daoMock[[]*dao.Layer]{resp: []*dao.Layer{manuscript}},

			expect: created,
		},
		{
			name: "Success/MostTags",

			request: withTags(tenLabels...),

			lockMock:    &daoMock[struct{}]{},
			countMock:   &daoMock[*dao.ProjectCount]{resp: &dao.ProjectCount{Count: core.ProjectsPerOwnerMax - 1}},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layersMock:  &daoMock[[]*dao.Layer]{resp: []*dao.Layer{manuscript}},

			expect: created,
		},
		{
			name: "Success/MostGenres",

			request: withGenres(tenLabels...),

			lockMock:    &daoMock[struct{}]{},
			countMock:   &daoMock[*dao.ProjectCount]{resp: &dao.ProjectCount{Count: core.ProjectsPerOwnerMax - 1}},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layersMock:  &daoMock[[]*dao.Layer]{resp: []*dao.Layer{manuscript}},

			expect: created,
		},
		{
			name: "Success/LongestTitle",

			request: request(strings.Repeat("é", 80), "fr"),

			lockMock:    &daoMock[struct{}]{},
			countMock:   &daoMock[*dao.ProjectCount]{resp: &dao.ProjectCount{Count: core.ProjectsPerOwnerMax - 1}},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layersMock:  &daoMock[[]*dao.Layer]{resp: []*dao.Layer{manuscript}},

			expect: created,
		},
		{
			name: "Error/BlankTitle",

			request: request("   ", "fr"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TitleTooLong",

			request: request(strings.Repeat("é", 81), "fr"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/UnknownLanguage",

			request: request("Clear Hours", "de"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyTags",

			request: withTags(append(slices.Clone(tenLabels), "k")...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyGenres",

			request: withGenres(append(slices.Clone(tenLabels), "k")...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateTags",

			request: withTags("noir", "noir"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateGenres",

			request: withGenres("noir", "noir"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/EmptyTag",

			request: withTags(""),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/EmptyGenre",

			request: withGenres(""),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/BlankTag",

			request: withTags("   "),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/BlankGenre",

			request: withGenres("   "),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagLeadingWhitespace",

			request: withTags(" noir"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/GenreLeadingWhitespace",

			request: withGenres(" noir"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagTrailingWhitespace",

			request: withTags("noir\n"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/GenreTrailingWhitespace",

			request: withGenres("noir\n"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagNUL",

			request: withTags("no\x00ir"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/GenreNUL",

			request: withGenres("no\x00ir"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagTooLong",

			request: withTags(strings.Repeat("é", 25)),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/GenreTooLong",

			request: withGenres(strings.Repeat("é", 25)),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagUppercase",

			request: withTags("Noir"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagSpace",

			request: withTags("huis clos"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagUnderscore",

			request: withTags("roman_maritime"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagLeadingHyphen",

			request: withTags("-noir"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagTrailingHyphen",

			request: withTags("noir-"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagDoubleHyphen",

			request: withTags("huis--clos"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TagDecomposedAccent",

			// The accent is a combining mark after the "e": the tag is not precomposed (NFC), where "années-50"
			// holds a precomposed "é".
			request: withTags("anne\u0301es-50"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectsFull",

			request: request("Les Heures claires", "fr"),

			lockMock:  &daoMock[struct{}]{},
			countMock: &daoMock[*dao.ProjectCount]{resp: &dao.ProjectCount{Count: core.ProjectsPerOwnerMax}},

			expectErr: core.ErrProjectsFull,
		},
		{
			name: "Error/LockOwner",

			request: request("Les Heures claires", "fr"),

			lockMock: &daoMock[struct{}]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/CountProjects",

			request: request("Les Heures claires", "fr"),

			lockMock:  &daoMock[struct{}]{},
			countMock: &daoMock[*dao.ProjectCount]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/InsertProject",

			request: request("Les Heures claires", "fr"),

			lockMock:    &daoMock[struct{}]{},
			countMock:   &daoMock[*dao.ProjectCount]{resp: &dao.ProjectCount{Count: core.ProjectsPerOwnerMax - 1}},
			projectMock: &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/InsertLayers",

			request: request("Les Heures claires", "fr"),

			lockMock:    &daoMock[struct{}]{},
			countMock:   &daoMock[*dao.ProjectCount]{resp: &dao.ProjectCount{Count: core.ProjectsPerOwnerMax - 1}},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			layersMock:  &daoMock[[]*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request:    request("Les Heures claires", "fr"),
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoLockMock := coremocks.NewMockProjectCreateDaoLock(t)
			daoCountMock := coremocks.NewMockProjectCreateDaoCount(t)
			daoProjectMock := coremocks.NewMockProjectCreateDaoProject(t)
			daoLayersMock := coremocks.NewMockProjectCreateDaoLayers(t)

			if testCase.lockMock != nil {
				daoLockMock.EXPECT().
					Exec(mock.Anything, &dao.OwnerLockRequest{OwnerID: ownerID}).
					Return(testCase.lockMock.err)
			}

			if testCase.countMock != nil {
				daoCountMock.EXPECT().
					Exec(mock.Anything, &dao.ProjectCountRequest{OwnerID: ownerID}).
					Return(testCase.countMock.resp, testCase.countMock.err)
			}

			if testCase.projectMock != nil {
				daoProjectMock.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectInsertRequest) bool {
						return req.ID != uuid.Nil && req.OwnerID == testCase.request.OwnerID &&
							req.Title == testCase.request.Title && req.Premise == testCase.request.Premise &&
							req.Language == testCase.request.Language && recent(req.Now) &&
							req.Tags != nil && slices.Equal(req.Tags, testCase.request.Tags) &&
							req.Genres != nil && slices.Equal(req.Genres, testCase.request.Genres)
					})).
					Return(testCase.projectMock.resp, testCase.projectMock.err)
			}

			if testCase.layersMock != nil {
				daoLayersMock.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayersInsertRequest) bool {
						ids := lo.Map(req.Layers, func(item dao.LayersInsertItem, _ int) uuid.UUID { return item.ID })
						layers := lo.Map(req.Layers, func(item dao.LayersInsertItem, _ int) dao.LayersInsertItem {
							item.ID = uuid.Nil

							return item
						})

						return req.ProjectID == projectID && req.Position == 0 && recent(req.Now) && freshIDs(ids) &&
							slices.Equal(layers, []dao.LayersInsertItem{{Kind: core.LayerKindManuscript, Name: "Manuscript"}})
					})).
					Return(testCase.layersMock.resp, testCase.layersMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewProjectCreate(daoLockMock, daoCountMock, daoProjectMock, daoLayersMock, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			daoLockMock.AssertExpectations(t)
			daoCountMock.AssertExpectations(t)
			daoProjectMock.AssertExpectations(t)
			daoLayersMock.AssertExpectations(t)
		})
	}
}
