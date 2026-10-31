package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

// genaiID names the genai generation of the generation tests.
var genaiID = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")

// suggestOutput is a valid suggest output, with one text suggestion.
const suggestOutput = `{"manuscript":[],"text":[{"target":"T1","note":"Names the scene.","text":"La lettre"}]}`

// concurrencyCap is the message genai refuses a call with once the user runs as many as their tier allows.
const concurrencyCap = `usage cap reached: purpose "studio.generation" concurrency: 2 generations already in flight`

// asRoles returns the request as the auth middleware hands it over for the project owner, granted roles.
func asRoles(request *http.Request, roles ...string) *http.Request {
	claims := &serviceauthentication.Claims{UserID: &ownerID, Roles: roles}

	return request.WithContext(serviceauthentication.SetClaimsContext(request.Context(), claims))
}

func TestRestGenerationSubmit(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPost, "/v1/generation", map[string]any{
			"projectID":   projectID,
			"kind":        "suggest",
			"targets":     []any{map[string]any{"sceneID": sceneID, "field": "title", "mode": "revise"}},
			"instruction": "Plus court.",
			"variant":     1,
			"stream":      true,
		})
	}

	context := &core.ContextCall{
		Targets: []*core.ContextTarget{{
			Handle: "M1", SceneID: sceneID, LayerID: layerID, Mode: core.TargetModeRevise,
			Range: &core.Range{Start: 0, End: 2}, Kept: []int{1},
			Blocks: []*core.ContextBlock{{Handle: "B1", Position: 0}, {Handle: "B2", Position: 1}}, UpdatedAt: updatedAt,
		}},
		Scenes: []*core.ContextScene{
			{Handle: "S1", SceneID: sceneID, Level: core.ContextLevelFull},
			{Handle: "S2", SceneID: uuid.MustParse("00000000-0000-0000-0000-000000000012"), Level: core.ContextLevelOmitted},
		},
		Layers:   []*core.ContextLayer{{Handle: "L1", LayerID: layerID}},
		Estimate: 1200,
	}
	generation := &core.Generation{Kind: core.GenerationKindSuggest, Calls: []*core.GenerationCall{
		{Result: &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusRunning}, Context: context},
		{Waiting: concurrencyCap, Context: context},
	}}

	jsonContext := map[string]any{
		"targets": []any{map[string]any{
			"handle": "M1", "sceneID": sceneID.String(), "layerID": layerID.String(), "beatID": nil, "mode": "revise",
			"range": map[string]any{"start": 0.0, "end": 2.0}, "kept": []any{1.0},
			"blocks": []any{
				map[string]any{"handle": "B1", "position": 0.0}, map[string]any{"handle": "B2", "position": 1.0},
			},
			"override": false, "updatedAt": updatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}},
		"scenes":   []any{map[string]any{"handle": "S1", "sceneID": sceneID.String(), "level": "full"}},
		"layers":   []any{map[string]any{"handle": "L1", "layerID": layerID.String()}},
		"estimate": 1200.0,
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		// admin is whether the service runs the generation on genai's admin tier.
		admin       bool
		serviceMock *serviceMock[*core.Generation]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.Generation]{resp: generation},

			expectStatus: http.StatusAccepted,
			expectResponse: map[string]any{"kind": "suggest", "calls": []any{
				map[string]any{
					"generation": map[string]any{"id": genaiID.String(), "status": "running"},
					"context":    jsonContext,
				},
				map[string]any{"generation": nil, "waiting": concurrencyCap, "context": jsonContext},
			}},
		},
		{
			name: "Success/Admin",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return asRoles(validRequest(t), "auth:admin")
			},

			admin:       true,
			serviceMock: &serviceMock[*core.Generation]{resp: &core.Generation{Kind: core.GenerationKindSuggest}},

			expectStatus:   http.StatusAccepted,
			expectResponse: map[string]any{"kind": "suggest", "calls": []any{}},
		},
		{
			// A superadmin inherits the admin's permission.
			name: "Success/Superadmin",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return asRoles(validRequest(t), "auth:superadmin")
			},

			admin:       true,
			serviceMock: &serviceMock[*core.Generation]{resp: &core.Generation{Kind: core.GenerationKindSuggest}},

			expectStatus: http.StatusAccepted,
		},
		{
			name: "Success/User",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return asRoles(validRequest(t), "auth:user")
			},

			serviceMock: &serviceMock[*core.Generation]{resp: &core.Generation{Kind: core.GenerationKindSuggest}},

			expectStatus: http.StatusAccepted,
		},
		{
			name: "Error/Anonymous",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return asUser(validRequest(t), nil)
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/MalformedRequest",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodPost, "/v1/generation", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/InvalidRequest",

			request: validRequest,

			serviceMock: &serviceMock[*core.Generation]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.Generation]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/ContextTooLarge",

			request: validRequest,

			serviceMock: &serviceMock[*core.Generation]{err: core.ErrContextTooLarge},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/Quota",

			request: validRequest,

			serviceMock: &serviceMock[*core.Generation]{err: &core.GenerationQuotaError{Cap: concurrencyCap}},

			expectStatus: http.StatusTooManyRequests,
			expectResponse: map[string]any{
				"type": "about:blank", "title": "Too Many Requests", "status": 429.0,
				"tags": map[string]any{"cap": concurrencyCap},
			},
		},
		{
			name: "Error/GenaiUnavailable",

			request: validRequest,

			serviceMock: &serviceMock[*core.Generation]{err: core.ErrGenaiUnavailable},

			expectStatus: http.StatusServiceUnavailable,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[*core.Generation]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestGenerationSubmitService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.GenerationSubmitRequest{
						OwnerID:   ownerID,
						ProjectID: projectID,
						Admin:     testCase.admin,
						Kind:      core.GenerationKindSuggest,
						Targets: []*core.GenerationTarget{
							{SceneID: sceneID, Field: core.SceneFieldTitle, Mode: core.TargetModeRevise},
						},
						Instruction: "Plus court.",
						Variant:     1,
						Stream:      true,
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestGenerationSubmit(service, config.PermissionsConfigDefault, config.LoggerDev).
				ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}

// suggestModel names the model that wrote [suggestOutput].
const suggestModel = "gpt-6.1-terra-2026-09-30"

// jsonResult is the JSON of a succeeded genai generation of [suggestOutput].
var jsonResult = map[string]any{
	"id": genaiID.String(), "status": "succeeded", "output": mustDecode(suggestOutput), "model": suggestModel,
}

// mustDecode decodes JSON, for an expected response body.
func mustDecode(data string) any {
	var value any

	err := json.Unmarshal([]byte(data), &value)
	if err != nil {
		panic(err)
	}

	return value
}
