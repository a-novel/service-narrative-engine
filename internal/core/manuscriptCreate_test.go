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
)

func TestManuscriptCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")
	stepValue, err := json.Marshal(manuscriptValue)
	require.NoError(t, err)

	validRequest := &core.ManuscriptCreateRequest{
		Actor:           core.Actor{UserID: ownerID},
		IdeaID:          ideaID,
		EngineVersionID: engineVersionID,
		StepKey:         "manuscript",
		StepValue:       stepValue,
		Manuscript:      manuscriptValue,
	}
	manuscriptID := uuid.MustParse("00000000-0000-0000-0000-000000000701")
	entity := &dao.Manuscript{
		ID:        manuscriptID,
		IdeaID:    ideaID,
		Value:     stepValue,
		CreatedAt: createdAt,
	}
	charactersStepValue := json.RawMessage(`{"characters":["Mara"]}`)
	charactersEngine := &dao.EngineVersion{
		ID: engineVersionID,
		Definition: json.RawMessage(`{
  "steps": [{
    "key": "characters",
    "promptTemplate": "List the main characters.",
    "outputSchema": {
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "type": "object",
      "additionalProperties": false,
      "required": ["characters"],
      "properties": {
        "characters": {
          "type": "array",
          "minItems": 1,
          "items": {"type": "string", "minLength": 1}
        }
      }
    }
  }]
}`),
	}

	testCases := []struct {
		name string

		request *core.ManuscriptCreateRequest

		ideaResponse   *dao.Idea
		ideaErr        error
		engineResponse *dao.EngineVersion
		engineErr      error
		stepErr        error
		callStep       bool
		manuscriptResp *dao.Manuscript
		manuscriptErr  error
		callManuscript bool
		transactionErr error

		expect             *core.Manuscript
		expectErr          error
		expectErrContains  string
		expectTransactions int
	}{
		{
			name:           "Success/WithoutGeneration",
			request:        validRequest,
			ideaResponse:   ideaFixture(),
			engineResponse: engineVersionFixture(),
			callStep:       true,
			manuscriptResp: entity,
			callManuscript: true,
			expect: &core.Manuscript{
				ID:        manuscriptID,
				IdeaID:    ideaID,
				Value:     manuscriptValue,
				CreatedAt: createdAt,
			},
			expectTransactions: 1,
		},
		{
			name: "Success/IndependentManuscriptContract",
			request: &core.ManuscriptCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "characters",
				StepValue:       charactersStepValue,
				Manuscript:      manuscriptValue,
			},
			ideaResponse:   ideaFixture(),
			engineResponse: charactersEngine,
			callStep:       true,
			manuscriptResp: entity,
			callManuscript: true,
			expect: &core.Manuscript{
				ID:        manuscriptID,
				IdeaID:    ideaID,
				Value:     manuscriptValue,
				CreatedAt: createdAt,
			},
			expectTransactions: 1,
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.ManuscriptCreateRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidTypedManuscript",
			request: &core.ManuscriptCreateRequest{
				Actor:           validRequest.Actor,
				IdeaID:          ideaID,
				EngineVersionID: engineVersionID,
				StepKey:         "manuscript",
				StepValue:       stepValue,
				Manuscript:      core.ManuscriptValue{},
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "Error/IdeaNotFound",
			request:   validRequest,
			ideaErr:   dao.ErrIdeaSelectNotFound,
			expectErr: core.ErrIdeaNotFound,
		},
		{
			name:      "Error/IdeaDao",
			request:   validRequest,
			ideaErr:   errFoo,
			expectErr: errFoo,
		},
		{
			name:         "Error/EngineVersionNotFound",
			request:      validRequest,
			ideaResponse: ideaFixture(),
			engineErr:    dao.ErrEngineVersionSelectNotFound,
			expectErr:    core.ErrEngineVersionNotFound,
		},
		{
			name:         "Error/EngineVersionDao",
			request:      validRequest,
			ideaResponse: ideaFixture(),
			engineErr:    errFoo,
			expectErr:    errFoo,
		},
		{
			name:           "Error/InvalidStepValue",
			request:        manuscriptRequestWithStepValue(json.RawMessage(`{"title":""}`)),
			ideaResponse:   ideaFixture(),
			engineResponse: engineVersionFixture(),
			expectErr:      core.ErrInvalidRequest,
		},
		{
			name:               "Error/StepConflict",
			request:            validRequest,
			ideaResponse:       ideaFixture(),
			engineResponse:     engineVersionFixture(),
			stepErr:            dao.ErrStepValueInsertConflict,
			callStep:           true,
			expectErr:          core.ErrStepValueConflict,
			expectTransactions: 1,
		},
		{
			name:               "Error/StepInsert",
			request:            validRequest,
			ideaResponse:       ideaFixture(),
			engineResponse:     engineVersionFixture(),
			stepErr:            errFoo,
			callStep:           true,
			expectErr:          errFoo,
			expectTransactions: 1,
		},
		{
			name:               "Error/ManuscriptInsert",
			request:            validRequest,
			ideaResponse:       ideaFixture(),
			engineResponse:     engineVersionFixture(),
			callStep:           true,
			manuscriptErr:      errFoo,
			callManuscript:     true,
			expectErr:          errFoo,
			expectTransactions: 1,
		},
		{
			name:               "Error/Transaction",
			request:            validRequest,
			ideaResponse:       ideaFixture(),
			engineResponse:     engineVersionFixture(),
			transactionErr:     errFoo,
			expectErr:          errFoo,
			expectTransactions: 1,
		},
		{
			name:               "Error/MissingManuscript",
			request:            validRequest,
			ideaResponse:       ideaFixture(),
			engineResponse:     engineVersionFixture(),
			callStep:           true,
			callManuscript:     true,
			expectErrContains:  "save project content",
			expectTransactions: 1,
		},
		{
			name:           "Error/InvalidStoredManuscript",
			request:        validRequest,
			ideaResponse:   ideaFixture(),
			engineResponse: engineVersionFixture(),
			callStep:       true,
			manuscriptResp: &dao.Manuscript{
				ID:     manuscriptID,
				IdeaID: ideaID,
				Value:  json.RawMessage(`{`),
			},
			callManuscript:     true,
			expectErrContains:  "decode saved Manuscript",
			expectTransactions: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ideaDao := coremocks.NewMockIdeaSelectDao(t)
			engineVersionDao := coremocks.NewMockEngineVersionSelectDao(t)
			stepValueDao := coremocks.NewMockStepValueInsertDao(t)
			manuscriptDao := coremocks.NewMockManuscriptInsertDao(t)

			if testCase.ideaResponse != nil || testCase.ideaErr != nil {
				ideaDao.EXPECT().
					Exec(mock.Anything, &dao.IdeaSelectRequest{
						ID:      testCase.request.IdeaID,
						OwnerID: testCase.request.Actor.UserID,
					}).
					Return(testCase.ideaResponse, testCase.ideaErr)
			}

			if testCase.engineResponse != nil || testCase.engineErr != nil {
				engineVersionDao.EXPECT().
					Exec(mock.Anything, &dao.EngineVersionSelectRequest{
						ID: testCase.request.EngineVersionID,
					}).
					Return(testCase.engineResponse, testCase.engineErr)
			}

			var saveTime time.Time

			if testCase.callStep {
				stepValueDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.StepValueInsertRequest) bool {
						saveTime = request.Now

						return assert.NotEqual(t, uuid.Nil, request.ID) &&
							assert.Equal(t, testCase.request.IdeaID, request.IdeaID) &&
							assert.Equal(t, testCase.request.EngineVersionID, request.EngineVersionID) &&
							assert.Equal(t, testCase.request.StepKey, request.StepKey) &&
							assert.JSONEq(t, string(testCase.request.StepValue), string(request.Value)) &&
							assert.WithinDuration(t, time.Now(), request.Now, time.Minute)
					})).
					Return(nil, testCase.stepErr)
			}

			if testCase.callManuscript {
				manuscriptDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.ManuscriptInsertRequest) bool {
						return assert.NotEqual(t, uuid.Nil, request.ID) &&
							assert.Equal(t, testCase.request.IdeaID, request.IdeaID) &&
							assert.Equal(t, saveTime, request.Now) &&
							assert.JSONEq(t, string(stepValue), string(request.Value))
					})).
					Return(testCase.manuscriptResp, testCase.manuscriptErr)
			}

			transactor := transactiontest.NewTransactor()
			if testCase.transactionErr != nil {
				transactor = transactiontest.NewFailingTransactor(testCase.transactionErr)
			}

			result, execErr := core.NewManuscriptCreate(
				ideaDao,
				engineVersionDao,
				stepValueDao,
				manuscriptDao,
				transactor,
			).Exec(t.Context(), testCase.request)

			if testCase.expectErr != nil {
				require.ErrorIs(t, execErr, testCase.expectErr)
			} else if testCase.expectErrContains == "" {
				require.NoError(t, execErr)
			} else {
				require.ErrorContains(t, execErr, testCase.expectErrContains)
			}

			require.Equal(t, testCase.expect, result)
			require.Equal(t, testCase.expectTransactions, transactor.Calls())
		})
	}
}

func manuscriptRequestWithStepValue(value json.RawMessage) *core.ManuscriptCreateRequest {
	return &core.ManuscriptCreateRequest{
		Actor:           core.Actor{UserID: ownerID},
		IdeaID:          ideaID,
		EngineVersionID: engineVersionID,
		StepKey:         "manuscript",
		StepValue:       value,
		Manuscript:      manuscriptValue,
	}
}
