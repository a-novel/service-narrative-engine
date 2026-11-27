package core_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/v2/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

func TestInstanceLinksUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	otherInstanceID := uuid.MustParse("00000000-0000-0000-0000-000000000032")

	link := core.Link{
		Source:       "scenes/a",
		Destination:  core.Identifier{ID: otherInstanceID, Target: "cast/b"},
		Relationship: "therefore",
	}

	// linkJSON is the link as the instance stores it.
	const linkJSON = `{"source":"scenes/a",` +
		`"destination":{"id":"00000000-0000-0000-0000-000000000032","target":"cast/b"},` +
		`"relationship":"therefore"}`

	withLinks := func(links ...core.Link) *core.InstanceLinksUpdateRequest {
		return &core.InstanceLinksUpdateRequest{
			OwnerID: ownerID, ProjectID: projectID, ID: instanceID, Links: links, UpdatedAt: new(documentUpdatedAt),
		}
	}

	// changed returns the link once change edited it.
	changed := func(change func(link *core.Link)) core.Link {
		edited := link
		change(&edited)

		return edited
	}

	// longest is a text of the most characters a target holds, each of two bytes.
	longest := strings.Repeat("é", caps.LinkTargetLength)

	written := &daoMock[*dao.InstanceLinksUpdateRow]{resp: &dao.InstanceLinksUpdateRow{UpdatedAt: editedAt}}

	testCases := []struct {
		name string

		request *core.InstanceLinksUpdateRequest

		lockMock   *daoMock[bool]
		daoMock    *daoMock[*dao.InstanceLinksUpdateRow]
		bytesMock  *daoMock[*dao.ProjectBytes]
		transactor transaction.Transactor

		// expectLinks is the document the write stores.
		expectLinks string
		expect      *core.WrittenInstanceLinks
		expectErr   error
		expectField *core.FieldError
	}{
		{
			name: "Success",

			request: withLinks(link),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   written,
			bytesMock: withinBudget,

			expectLinks: "[" + linkJSON + "]",
			expect:      &core.WrittenInstanceLinks{UpdatedAt: editedAt},
		},
		{
			// An instance that never wrote its links names no instant.
			name: "Success/FirstWrite",

			request: &core.InstanceLinksUpdateRequest{
				OwnerID: ownerID, ProjectID: projectID, ID: instanceID, Links: []core.Link{link},
			},

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   written,
			bytesMock: withinBudget,

			expectLinks: "[" + linkJSON + "]",
			expect:      &core.WrittenInstanceLinks{UpdatedAt: editedAt},
		},
		{
			// An empty list clears the links, where a missing one is refused.
			name: "Success/Clear",

			request: &core.InstanceLinksUpdateRequest{
				OwnerID: ownerID, ProjectID: projectID, ID: instanceID, Links: []core.Link{},
				UpdatedAt: new(documentUpdatedAt),
			},

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   written,
			bytesMock: withinBudget,

			expectLinks: "[]",
			expect:      &core.WrittenInstanceLinks{UpdatedAt: editedAt},
		},
		{
			// A target counts its characters, a relationship its bytes, and a link may leave its relationship
			// empty.
			name: "Success/MostOfEach",

			request: withLinks(
				changed(func(link *core.Link) { link.Source = longest }),
				changed(func(link *core.Link) { link.Relationship = strings.Repeat("a", caps.LinkRelationshipBytes) }),
				changed(func(link *core.Link) { link.Relationship = "" }),
			),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   written,
			bytesMock: withinBudget,

			expectLinks: "[" +
				strings.Replace(linkJSON, "scenes/a", longest, 1) + "," +
				strings.Replace(linkJSON, "therefore", strings.Repeat("a", caps.LinkRelationshipBytes), 1) + "," +
				strings.Replace(linkJSON, "therefore", "", 1) + "]",
			expect: &core.WrittenInstanceLinks{UpdatedAt: editedAt},
		},
		{
			// An omitted list never wipes the links.
			name: "Error/NoLinks",

			request: &core.InstanceLinksUpdateRequest{OwnerID: ownerID, ProjectID: projectID, ID: instanceID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoInstance",

			request: &core.InstanceLinksUpdateRequest{OwnerID: ownerID, ProjectID: projectID, Links: []core.Link{link}},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoSource",

			request: withLinks(changed(func(link *core.Link) { link.Source = "" })),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoDestinationID",

			request: withLinks(changed(func(link *core.Link) { link.Destination.ID = uuid.Nil })),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoDestinationTarget",

			request: withLinks(changed(func(link *core.Link) { link.Destination.Target = "" })),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyLinks",

			request: withLinks(link, link, link, link),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "links", Rule: "max"},
		},
		{
			name: "Error/SourceTooLong",

			request: withLinks(link, changed(func(link *core.Link) { link.Source = longest + "é" })),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "links[1].source", Rule: "max"},
		},
		{
			name: "Error/DestinationTargetTooLong",

			request: withLinks(changed(func(link *core.Link) { link.Destination.Target = longest + "é" })),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "links[0].destination.target", Rule: "max"},
		},
		{
			name: "Error/RelationshipTooLarge",

			request: withLinks(changed(func(link *core.Link) {
				link.Relationship = strings.Repeat("a", caps.LinkRelationshipBytes+1)
			})),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "links[0].relationship", Rule: "maxbytes"},
		},
		{
			// Three links whose targets hold the most characters weigh 436 bytes, one past the most the links of
			// an instance hold.
			name: "Error/TooLarge",

			request: withLinks(
				changed(func(link *core.Link) { link.Source, link.Destination.Target = longest, longest }),
				changed(func(link *core.Link) { link.Source, link.Destination.Target = longest, longest }),
				changed(func(link *core.Link) { link.Source, link.Destination.Target = longest, longest }),
			),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "links", Rule: "maxbytes"},
		},
		{
			name: "Error/Nul",

			request: withLinks(changed(func(link *core.Link) { link.Relationship = "a\x00b" })),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "links", Rule: "nonul"},
		},
		{
			name: "Error/NotOwned",

			request: withLinks(link),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Stale",

			request: withLinks(link),

			lockMock:    &daoMock[bool]{resp: true},
			daoMock:     &daoMock[*dao.InstanceLinksUpdateRow]{err: dao.ErrInstanceLinksUpdateStale},
			expectLinks: "[" + linkJSON + "]",

			expectErr: core.ErrInstanceLinksStale,
		},
		{
			name: "Error/InstanceNotFound",

			request: withLinks(link),

			lockMock:    &daoMock[bool]{resp: true},
			daoMock:     &daoMock[*dao.InstanceLinksUpdateRow]{err: dao.ErrInstanceNotFound},
			expectLinks: "[" + linkJSON + "]",

			expectErr: core.ErrInstanceNotFound,
		},
		{
			name: "Error/Internal",

			request: withLinks(link),

			lockMock:    &daoMock[bool]{resp: true},
			daoMock:     &daoMock[*dao.InstanceLinksUpdateRow]{err: errFoo},
			expectLinks: "[" + linkJSON + "]",

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: withLinks(link),

			lockMock:    &daoMock[bool]{resp: true},
			daoMock:     written,
			bytesMock:   overBudget,
			expectLinks: "[" + linkJSON + "]",

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/Transaction",

			request: withLinks(link),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockProjectLockDao(t)
			mockDao := coremocks.NewMockInstanceLinksUpdateDao(t)
			mockBytes := coremocks.NewMockProjectBytesDao(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.InstanceLinksUpdateRequest) bool {
						return req.ProjectID == projectID && req.ID == instanceID &&
							string(req.Links) == testCase.expectLinks &&
							sameInstant(req.UpdatedAt, testCase.request.UpdatedAt) && recent(req.Now)
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

			service := core.NewInstanceLinksUpdate(mockLock, mockDao, mockBytes, transactor, caps)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			requireFieldError(t, testCase.expectField, err)
			require.Equal(t, testCase.expect, result)
		})
	}
}

// sameInstant reports whether two optional instants are both absent, or name the same one.
func sameInstant(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Equal(*b)
}
