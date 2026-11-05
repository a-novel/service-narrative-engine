package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestGenerationUsage(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return asRoles(newRequest(t, http.MethodGet, "/v1/generation/usage", nil), "auth:user")
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		// roles are the caller's roles the service receives, which set their caps.
		roles       []string
		serviceMock *serviceMock[*core.GenerationUsageSummary]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			roles: []string{"auth:user"},
			serviceMock: &serviceMock[*core.GenerationUsageSummary]{resp: &core.GenerationUsageSummary{
				Caps: &core.GenerationCaps{
					Concurrency: lo.ToPtr[int32](2),
					Windows: []*core.GenerationCapWindow{
						{Duration: 5 * time.Hour, InputTokens: lo.ToPtr[int64](2000), OutputTokens: lo.ToPtr[int64](200)},
						{Duration: 720 * time.Hour, OutputTokens: lo.ToPtr[int64](5000)},
					},
				},
				InFlight: 1,
				Windows: []*core.GenerationUsageWindow{
					{Duration: 5 * time.Hour, InputTokens: 1200, OutputTokens: 30},
					{Duration: 720 * time.Hour, InputTokens: 9000, OutputTokens: 400},
				},
			}},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"caps": map[string]any{
					"concurrency": 2.0,
					"windows": []any{
						map[string]any{"durationSeconds": 18000.0, "inputTokens": 2000.0, "outputTokens": 200.0},
						map[string]any{"durationSeconds": 2592000.0, "outputTokens": 5000.0},
					},
				},
				"inFlight": 1.0,
				"windows": []any{
					map[string]any{"durationSeconds": 18000.0, "inputTokens": 1200.0, "outputTokens": 30.0},
					map[string]any{"durationSeconds": 2592000.0, "inputTokens": 9000.0, "outputTokens": 400.0},
				},
			},
		},
		{
			// An uncapped caller's empty caps limit nothing, and leave no window to report.
			name: "Success/Uncapped",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return asRoles(newRequest(t, http.MethodGet, "/v1/generation/usage", nil), "auth:admin")
			},

			roles: []string{"auth:admin"},
			serviceMock: &serviceMock[*core.GenerationUsageSummary]{resp: &core.GenerationUsageSummary{
				Caps: &core.GenerationCaps{}, InFlight: 3,
			}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"caps": map[string]any{}, "inFlight": 3.0, "windows": []any{}},
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
			name: "Error/NoCaps",

			request: validRequest,

			roles:       []string{"auth:user"},
			serviceMock: &serviceMock[*core.GenerationUsageSummary]{err: core.ErrNoGenerationCaps},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/GenaiUnavailable",

			request: validRequest,

			roles:       []string{"auth:user"},
			serviceMock: &serviceMock[*core.GenerationUsageSummary]{err: core.ErrGenaiUnavailable},

			expectStatus: http.StatusServiceUnavailable,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			roles:       []string{"auth:user"},
			serviceMock: &serviceMock[*core.GenerationUsageSummary]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestGenerationUsageService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.GenerationUsageRequest{OwnerID: ownerID, Roles: testCase.roles}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestGenerationUsage(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
