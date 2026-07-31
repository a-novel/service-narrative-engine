package core_test

import (
	"encoding/json"
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
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

func TestStepValueCreate(t *testing.T) {
	t.Parallel()

	const privateSchemaValue = "do-not-trace-schema-value"

	errFoo := errors.New("foo")
	partialValue := json.RawMessage(`{"title":"A partial proposal"}`)
	validRequest := &core.StepValueCreateRequest{
		Actor:           core.Actor{UserID: ownerID},
		IdeaID:          ideaID,
		EngineVersionID: engineVersionID,
		StepKey:         "manuscript",
		Value:           partialValue,
	}
	entity := &dao.StepValue{
		ID:              uuid.MustParse("00000000-0000-0000-0000-000000000801"),
		IdeaID:          ideaID,
		EngineVersionID: engineVersionID,
		StepKey:         "manuscript",
		Value:           partialValue,
		CreatedAt:       createdAt,
	}
	validationEngineVersion := validationEngineVersionFixture()
	documentJustBelowLimit := contentDocumentOfSize(schemas.ContentDocumentMaxBytes - 1)
	documentAtLimit := contentDocumentOfSize(schemas.ContentDocumentMaxBytes)
	documentOverLimit := contentDocumentOfSize(schemas.ContentDocumentMaxBytes + 1)
	require.Len(t, documentJustBelowLimit, schemas.ContentDocumentMaxBytes-1)
	require.Len(t, documentAtLimit, schemas.ContentDocumentMaxBytes)
	require.Len(t, documentOverLimit, schemas.ContentDocumentMaxBytes+1)

	testCases := []struct {
		name string

		request *core.StepValueCreateRequest

		accessErr      error
		callAccess     bool
		engineResponse *dao.EngineVersion
		engineErr      error
		callEngine     bool
		daoResponse    *dao.StepValue
		daoErr         error
		callDao        bool
		transactorErr  error

		expect            json.RawMessage
		expectErr         error
		expectErrMessage  string
		expectErrExcludes string
	}{
		{
			name:           "Success/DeepPartial",
			request:        validRequest,
			callAccess:     true,
			engineResponse: engineVersionFixture(),
			callEngine:     true,
			daoResponse:    entity,
			callDao:        true,
			expect:         partialValue,
		},
		{
			name: "Success/EmptyPartial",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "manuscript",
				Value:           json.RawMessage(`{}`),
			},
			callAccess:     true,
			engineResponse: engineVersionFixture(),
			callEngine:     true,
			daoResponse: &dao.StepValue{
				ID:              entity.ID,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "manuscript",
				Value:           json.RawMessage(`{}`),
			},
			callDao: true,
			expect:  json.RawMessage(`{}`),
		},
		{
			name: "Success/DocumentJustBelowLimit",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "document",
				Value:           documentJustBelowLimit,
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			daoResponse: &dao.StepValue{
				ID:              entity.ID,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "document",
				Value:           documentJustBelowLimit,
			},
			callDao: true,
			expect:  documentJustBelowLimit,
		},
		{
			name: "Success/DocumentAtLimit",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "document",
				Value:           documentAtLimit,
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			daoResponse: &dao.StepValue{
				ID:              entity.ID,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "document",
				Value:           documentAtLimit,
			},
			callDao: true,
			expect:  documentAtLimit,
		},
		{
			name: "Success/SchemaKeywordPropertyAndNegativeRule",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-walker",
				Value:           json.RawMessage(`{"required":"kept"}`),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			daoResponse: &dao.StepValue{
				ID:              entity.ID,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-walker",
				Value:           json.RawMessage(`{"required":"kept"}`),
			},
			callDao: true,
			expect:  json.RawMessage(`{"required":"kept"}`),
		},
		{
			name: "Success/DependentRequiredOptional",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-walker",
				Value:           json.RawMessage(`{"trigger":"set"}`),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			daoResponse: &dao.StepValue{
				ID:              entity.ID,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-walker",
				Value:           json.RawMessage(`{"trigger":"set"}`),
			},
			callDao: true,
			expect:  json.RawMessage(`{"trigger":"set"}`),
		},
		{
			name: "Success/ArrayDependencyOptional",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-walker",
				Value:           json.RawMessage(`{"legacy":"set"}`),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			daoResponse: &dao.StepValue{
				ID:              entity.ID,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-walker",
				Value:           json.RawMessage(`{"legacy":"set"}`),
			},
			callDao: true,
			expect:  json.RawMessage(`{"legacy":"set"}`),
		},
		{
			name: "Success/PartialOneOfMayMatchMultipleCompleteBranches",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"selection":{}}`),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			daoResponse: &dao.StepValue{
				ID:              entity.ID,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"selection":{}}`),
			},
			callDao: true,
			expect:  json.RawMessage(`{"selection":{}}`),
		},
		{
			name: "Success/ReferencedPartialOneOfMayMatchMultipleCompleteBranches",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"referencedSelection":{}}`),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			daoResponse: &dao.StepValue{
				ID:              entity.ID,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"referencedSelection":{}}`),
			},
			callDao: true,
			expect:  json.RawMessage(`{"referencedSelection":{}}`),
		},
		{
			name: "Success/AnchoredPartialOneOfMayMatchMultipleCompleteBranches",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"anchoredSelection":{}}`),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			daoResponse: &dao.StepValue{
				ID:              entity.ID,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"anchoredSelection":{}}`),
			},
			callDao: true,
			expect:  json.RawMessage(`{"anchoredSelection":{}}`),
		},
		{
			name: "Success/NestedResourcePartialOneOfMayMatchMultipleCompleteBranches",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"nestedResourceSelection":{}}`),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			daoResponse: &dao.StepValue{
				ID:              entity.ID,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"nestedResourceSelection":{}}`),
			},
			callDao: true,
			expect:  json.RawMessage(`{"nestedResourceSelection":{}}`),
		},
		{
			name: "Error/ScalarOneOfRemainsExclusive",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"numeric":1}`),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			expectErr:      core.ErrInvalidRequest,
		},
		{
			name: "Error/ReferencedScalarOneOfRemainsExclusive",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"referencedNumeric":1}`),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			expectErr:      core.ErrInvalidRequest,
		},
		{
			name: "Error/DeclarationOnlyPresenceDoesNotWeakenOneOf",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"declaredNumeric":1}`),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			expectErr:      core.ErrInvalidRequest,
		},
		{
			name: "Error/CyclicReferencesTerminateWithoutWeakeningOneOf",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-one-of",
				Value:           json.RawMessage(`{"cyclicSelection":{}}`),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			expectErr:      core.ErrInvalidRequest,
		},
		{
			name: "Error/SchemaDependencyStillValidatesShape",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "schema-walker",
				Value: json.RawMessage(
					`{"schemaTrigger":"set","secondary":42}`,
				),
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			expectErr:      core.ErrInvalidRequest,
		},
		{
			name: "Error/SchemaValuePrivacy",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "manuscript",
				Value: json.RawMessage(
					`{"format":"` + privateSchemaValue + `"}`,
				),
			},
			callAccess:        true,
			engineResponse:    engineVersionFixture(),
			callEngine:        true,
			expectErr:         core.ErrInvalidRequest,
			expectErrExcludes: privateSchemaValue,
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.StepValueCreateRequest{},
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
			name:       "Error/EngineVersionNotFound",
			request:    validRequest,
			callAccess: true,
			engineErr:  dao.ErrEngineVersionSelectNotFound,
			callEngine: true,
			expectErr:  core.ErrEngineVersionNotFound,
		},
		{
			name: "Error/Shape",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "manuscript",
				Value:           json.RawMessage(`{"unknown":true}`),
			},
			callAccess:     true,
			engineResponse: engineVersionFixture(),
			callEngine:     true,
			expectErr:      core.ErrInvalidRequest,
		},
		{
			name: "Error/DocumentOverLimit",
			request: &core.StepValueCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "document",
				Value:           documentOverLimit,
			},
			callAccess:     true,
			engineResponse: validationEngineVersion,
			callEngine:     true,
			expectErr:      core.ErrInvalidRequest,
		},
		{
			name:           "Error/OwnerRelock",
			request:        validRequest,
			callAccess:     true,
			engineResponse: engineVersionFixture(),
			callEngine:     true,
			daoErr:         dao.ErrIdeaLockNotFound,
			callDao:        true,
			expectErr:      core.ErrIdeaNotFound,
		},
		{
			name:           "Error/Insert",
			request:        validRequest,
			callAccess:     true,
			engineResponse: engineVersionFixture(),
			callEngine:     true,
			daoErr:         errFoo,
			callDao:        true,
			expectErr:      errFoo,
		},
		{
			name:           "Error/Transaction",
			request:        validRequest,
			callAccess:     true,
			engineResponse: engineVersionFixture(),
			callEngine:     true,
			transactorErr:  errFoo,
			expectErr:      errFoo,
		},
		{
			name:             "Error/MissingEntity",
			request:          validRequest,
			callAccess:       true,
			engineResponse:   engineVersionFixture(),
			callEngine:       true,
			callDao:          true,
			expectErrMessage: "insert step value",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectAccess := coremocks.NewMockProjectAccessService(t)
			engineVersionDao := coremocks.NewMockEngineVersionSelectDao(t)
			stepValueDao := coremocks.NewMockStepValueInsertDao(t)

			if testCase.callAccess {
				projectAccess.EXPECT().
					Exec(mock.Anything, &core.ProjectAccessRequest{
						Actor:  testCase.request.Actor,
						IdeaID: testCase.request.IdeaID,
					}).
					Return(ideaFixture(), testCase.accessErr)
			}

			if testCase.callEngine {
				engineVersionDao.EXPECT().
					Exec(mock.Anything, &dao.EngineVersionSelectRequest{
						ID: testCase.request.EngineVersionID,
					}).
					Return(testCase.engineResponse, testCase.engineErr)
			}

			if testCase.callDao {
				stepValueDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.StepValueInsertRequest) bool {
						return assert.NotEqual(t, uuid.Nil, request.ID) &&
							assert.Equal(t, testCase.request.IdeaID, request.IdeaID) &&
							assert.Equal(t, testCase.request.Actor.UserID, request.OwnerID) &&
							assert.Equal(t, testCase.request.EngineVersionID, request.EngineVersionID) &&
							assert.Equal(t, testCase.request.StepKey, request.StepKey) &&
							assert.JSONEq(t, string(testCase.request.Value), string(request.Value)) &&
							assert.WithinDuration(t, time.Now(), request.Now, time.Minute)
					})).
					Return(testCase.daoResponse, testCase.daoErr)
			}

			transactor := transactiontest.NewTransactor()
			if testCase.transactorErr != nil {
				transactor = transactiontest.NewFailingTransactor(testCase.transactorErr)
			}

			result, err := core.NewStepValueCreate(
				projectAccess,
				engineVersionDao,
				stepValueDao,
				transactor,
			).Exec(t.Context(), testCase.request)

			if testCase.expectErr != nil {
				require.ErrorIs(t, err, testCase.expectErr)
			} else if testCase.expectErrMessage != "" {
				require.ErrorContains(t, err, testCase.expectErrMessage)
			} else {
				require.NoError(t, err)
			}

			if testCase.expectErrExcludes != "" {
				require.NotContains(t, err.Error(), testCase.expectErrExcludes)
			}

			if testCase.expect == nil {
				require.Nil(t, result)
			} else {
				require.JSONEq(t, string(testCase.expect), string(result))
			}

			projectAccess.AssertExpectations(t)
			engineVersionDao.AssertExpectations(t)
			stepValueDao.AssertExpectations(t)
		})
	}
}
