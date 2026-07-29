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

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestManuscriptCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")
	partialManuscript := json.RawMessage(`{"format":"novel"}`)
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
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NonObject",
			request: &core.ManuscriptCreateRequest{
				Actor:      validRequest.Actor,
				IdeaID:     ideaID,
				Manuscript: json.RawMessage(`[]`),
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Shape",
			request: &core.ManuscriptCreateRequest{
				Actor:      validRequest.Actor,
				IdeaID:     ideaID,
				Manuscript: json.RawMessage(`{"unknown":true}`),
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DocumentTooLarge",
			request: &core.ManuscriptCreateRequest{
				Actor:  validRequest.Actor,
				IdeaID: ideaID,
				Manuscript: json.RawMessage(
					`{"format":"` + strings.Repeat("n", 5*1024*1024) + `"}`,
				),
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:       "Error/ProjectAccess",
			request:    validRequest,
			accessErr:  core.ErrIdeaNotFound,
			callAccess: true,
			expectErr:  core.ErrIdeaNotFound,
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
							assert.JSONEq(t, string(testCase.request.Manuscript), string(request.Value)) &&
							assert.WithinDuration(t, time.Now(), request.Now, time.Minute)
					})).
					Return(testCase.manuscriptResp, testCase.manuscriptErr)
			}

			result, err := core.NewManuscriptCreate(projectAccess, manuscriptDao).
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
		})
	}
}
