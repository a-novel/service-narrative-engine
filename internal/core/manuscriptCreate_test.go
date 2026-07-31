package core_test

import (
	"encoding/json"
	"errors"
	"strings"
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
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

func TestManuscriptCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")
	partialManuscript := json.RawMessage(
		`{"blocks":[{"type":"text",` +
			`"metadata":{"source":"draft","plugin":{"enabled":true,` +
			`"tags":["one",2,null]}},` +
			`"data":{"text":"Draft"}}]}`,
	)
	maxTextBlock := json.RawMessage(
		`{"blocks":[{"type":"text","metadata":{},"data":{"text":"` +
			strings.Repeat("n", 32*1024) +
			`","marks":[]}}]}`,
	)
	unicodeMarkedManuscript := json.RawMessage(
		`{"blocks":[{"type":"text","metadata":{},"data":{` +
			`"text":"é🙂界","marks":[` +
			`{"type":"bold","start":0,"end":3},` +
			`{"type":"italic","start":1,"end":2}]}}]}`,
	)
	invalidUTF8Manuscript := json.RawMessage{0xff}
	validRequest := &core.ManuscriptCreateRequest{
		Actor:      core.Actor{UserID: ownerID},
		IdeaID:     ideaID,
		Manuscript: partialManuscript,
	}
	manuscriptID := uuid.MustParse("00000000-0000-0000-0000-000000000701")
	entity := &dao.Manuscript{
		ID:        manuscriptID,
		IdeaID:    ideaID,
		Value:     partialManuscript,
		CreatedAt: createdAt,
	}

	testCases := []struct {
		name string

		request *core.ManuscriptCreateRequest

		accessErr        error
		callAccess       bool
		manuscriptResp   *dao.Manuscript
		manuscriptErr    error
		callManuscript   bool
		transactorErr    error
		expect           json.RawMessage
		expectErr        error
		expectErrMessage string
	}{
		{
			name:           "Success/PartialOpaqueDocument",
			request:        validRequest,
			callAccess:     true,
			manuscriptResp: entity,
			callManuscript: true,
			expect:         partialManuscript,
		},
		{
			name: "Success/UnicodeAndOverlappingMarks",
			request: &core.ManuscriptCreateRequest{
				Actor:      validRequest.Actor,
				IdeaID:     ideaID,
				Manuscript: unicodeMarkedManuscript,
			},
			callAccess: true,
			manuscriptResp: &dao.Manuscript{
				ID:     manuscriptID,
				IdeaID: ideaID,
				Value:  unicodeMarkedManuscript,
			},
			callManuscript: true,
			expect:         unicodeMarkedManuscript,
		},
		{
			name: "Success/TextBlockAtLimit",
			request: &core.ManuscriptCreateRequest{
				Actor:      validRequest.Actor,
				IdeaID:     ideaID,
				Manuscript: maxTextBlock,
			},
			callAccess: true,
			manuscriptResp: &dao.Manuscript{
				ID:     manuscriptID,
				IdeaID: ideaID,
				Value:  maxTextBlock,
			},
			callManuscript: true,
			expect:         maxTextBlock,
		},
		{
			name: "Success/EmptyPartialDocument",
			request: &core.ManuscriptCreateRequest{
				Actor:      validRequest.Actor,
				IdeaID:     ideaID,
				Manuscript: json.RawMessage(`{}`),
			},
			callAccess: true,
			manuscriptResp: &dao.Manuscript{
				ID:     manuscriptID,
				IdeaID: ideaID,
				Value:  json.RawMessage(`{}`),
			},
			callManuscript: true,
			expect:         json.RawMessage(`{}`),
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.ManuscriptCreateRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidJSON",
			request: &core.ManuscriptCreateRequest{
				Actor:      validRequest.Actor,
				IdeaID:     ideaID,
				Manuscript: json.RawMessage(`{`),
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidUTF8",
			request: &core.ManuscriptCreateRequest{
				Actor:      validRequest.Actor,
				IdeaID:     ideaID,
				Manuscript: invalidUTF8Manuscript,
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/NonObject",
			request: &core.ManuscriptCreateRequest{
				Actor:      validRequest.Actor,
				IdeaID:     ideaID,
				Manuscript: json.RawMessage(`[]`),
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/UnknownDocumentField",
			request: &core.ManuscriptCreateRequest{
				Actor:      validRequest.Actor,
				IdeaID:     ideaID,
				Manuscript: json.RawMessage(`{"unknown":true}`),
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/LegacyBlockShape",
			request: &core.ManuscriptCreateRequest{
				Actor:  validRequest.Actor,
				IdeaID: ideaID,
				Manuscript: json.RawMessage(
					`{"blocks":[{"type":"text","text":"legacy","marks":[]}]}`,
				),
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/TextBlockTooLong",
			request: &core.ManuscriptCreateRequest{
				Actor:  validRequest.Actor,
				IdeaID: ideaID,
				Manuscript: json.RawMessage(
					`{"blocks":[{"type":"text","metadata":{},"data":{"text":"` +
						strings.Repeat("n", 32*1024+1) +
						`","marks":[]}}]}`,
				),
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/EmptyMarkRange",
			request: &core.ManuscriptCreateRequest{
				Actor:  validRequest.Actor,
				IdeaID: ideaID,
				Manuscript: json.RawMessage(
					`{"blocks":[{"type":"text","metadata":{},"data":{` +
						`"text":"word","marks":[` +
						`{"type":"bold","start":1,"end":1}]}}]}`,
				),
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/ReversedMarkRange",
			request: &core.ManuscriptCreateRequest{
				Actor:  validRequest.Actor,
				IdeaID: ideaID,
				Manuscript: json.RawMessage(
					`{"blocks":[{"type":"text","metadata":{},"data":{` +
						`"text":"word","marks":[` +
						`{"type":"bold","start":3,"end":2}]}}]}`,
				),
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/MarkPastUnicodeText",
			request: &core.ManuscriptCreateRequest{
				Actor:  validRequest.Actor,
				IdeaID: ideaID,
				Manuscript: json.RawMessage(
					`{"blocks":[{"type":"text","metadata":{},"data":{` +
						`"text":"é🙂界","marks":[` +
						`{"type":"bold","start":0,"end":4}]}}]}`,
				),
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/LinkMark",
			request: &core.ManuscriptCreateRequest{
				Actor:  validRequest.Actor,
				IdeaID: ideaID,
				Manuscript: json.RawMessage(
					`{"blocks":[{"type":"text","metadata":{},"data":{` +
						`"text":"linked","marks":[` +
						`{"type":"link","start":0,"end":6}]}}]}`,
				),
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/MediaBlock",
			request: &core.ManuscriptCreateRequest{
				Actor:  validRequest.Actor,
				IdeaID: ideaID,
				Manuscript: json.RawMessage(
					`{"blocks":[{"type":"media","metadata":{},` +
						`"data":{"text":"image","marks":[]}}]}`,
				),
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/DocumentTooLarge",
			request: &core.ManuscriptCreateRequest{
				Actor:  validRequest.Actor,
				IdeaID: ideaID,
				Manuscript: json.RawMessage(
					`{"blocks":[{"type":"text","metadata":{},"data":{"text":"` +
						strings.Repeat("n", schemas.ContentDocumentMaxBytes) +
						`","marks":[]}}]}`,
				),
			},
			callAccess: true,
			expectErr:  core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectAccessBeforeContentValidation",
			request: &core.ManuscriptCreateRequest{
				Actor:      validRequest.Actor,
				IdeaID:     ideaID,
				Manuscript: json.RawMessage(`{`),
			},
			accessErr:  core.ErrIdeaNotFound,
			callAccess: true,
			expectErr:  core.ErrIdeaNotFound,
		},
		{
			name:           "Error/OwnerRelock",
			request:        validRequest,
			callAccess:     true,
			manuscriptErr:  dao.ErrIdeaLockNotFound,
			callManuscript: true,
			expectErr:      core.ErrIdeaNotFound,
		},
		{
			name:           "Error/Insert",
			request:        validRequest,
			callAccess:     true,
			manuscriptErr:  errFoo,
			callManuscript: true,
			expectErr:      errFoo,
		},
		{
			name:          "Error/Transaction",
			request:       validRequest,
			callAccess:    true,
			transactorErr: errFoo,
			expectErr:     errFoo,
		},
		{
			name:             "Error/MissingEntity",
			request:          validRequest,
			callAccess:       true,
			callManuscript:   true,
			expectErrMessage: "insert Manuscript",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectAccess := coremocks.NewMockProjectAccessService(t)
			manuscriptDao := coremocks.NewMockManuscriptInsertDao(t)

			if testCase.callAccess {
				projectAccess.EXPECT().
					Exec(mock.Anything, &core.ProjectAccessRequest{
						Actor:  testCase.request.Actor,
						IdeaID: testCase.request.IdeaID,
					}).
					Return(ideaFixture(), testCase.accessErr)
			}

			if testCase.callManuscript {
				manuscriptDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.ManuscriptInsertRequest) bool {
						return assert.NotEqual(t, uuid.Nil, request.ID) &&
							assert.Equal(t, testCase.request.IdeaID, request.IdeaID) &&
							assert.Equal(t, testCase.request.Actor.UserID, request.OwnerID) &&
							assert.JSONEq(t, string(testCase.request.Manuscript), string(request.Value)) &&
							assert.WithinDuration(t, time.Now(), request.Now, time.Minute)
					})).
					Return(testCase.manuscriptResp, testCase.manuscriptErr)
			}

			transactor := transactiontest.NewTransactor()
			if testCase.transactorErr != nil {
				transactor = transactiontest.NewFailingTransactor(testCase.transactorErr)
			}

			result, err := core.NewManuscriptCreate(projectAccess, manuscriptDao, transactor).
				Exec(t.Context(), testCase.request)

			if testCase.expectErr != nil {
				require.ErrorIs(t, err, testCase.expectErr)
			} else if testCase.expectErrMessage != "" {
				require.ErrorContains(t, err, testCase.expectErrMessage)
			} else {
				require.NoError(t, err)
			}

			if testCase.expect == nil {
				require.Nil(t, result)
			} else {
				require.JSONEq(t, string(testCase.expect), string(result))
			}

			projectAccess.AssertExpectations(t)
			manuscriptDao.AssertExpectations(t)
		})
	}
}
