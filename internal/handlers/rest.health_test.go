package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	servicegenai "github.com/a-novel/service-genai/pkg/go"
	servicegenaimocks "github.com/a-novel/service-genai/pkg/go/mocks"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
	servicejsonkeysmocks "github.com/a-novel/service-json-keys/v2/pkg/go/mocks"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

func TestRestHealth(t *testing.T) {
	t.Parallel()

	errJSONKeys := errors.New("json keys unavailable")
	errGenAI := errors.New("genai unavailable")

	type jsonKeysMock struct {
		response *servicejsonkeys.StatusResponse
		err      error
	}

	type genaiMock struct {
		response *servicegenai.StatusResponse
		err      error
	}

	testCases := []struct {
		name string

		jsonKeysMock jsonKeysMock
		genaiMock    genaiMock
		skipPostgres bool
		requestCount int

		expectResponse any
	}{
		{
			name: "Success/CachedBurst",
			jsonKeysMock: jsonKeysMock{
				response: &servicejsonkeys.StatusResponse{},
			},
			genaiMock: genaiMock{
				response: &servicegenai.StatusResponse{
					Queue: &servicegenai.QueueDepth{
						Pending:                 3,
						OldestPendingAgeSeconds: 90,
					},
				},
			},
			requestCount: 8,
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusUp},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:genai": map[string]any{
					"status": handlers.RestHealthStatusUp,
					"queue": map[string]any{
						"pending":          float64(3),
						"oldestPendingAge": "1m30s",
					},
				},
			},
		},
		{
			name: "Success/EmptyQueue",
			jsonKeysMock: jsonKeysMock{
				response: &servicejsonkeys.StatusResponse{},
			},
			genaiMock: genaiMock{
				response: &servicegenai.StatusResponse{
					Queue: &servicegenai.QueueDepth{},
				},
			},
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusUp},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:genai": map[string]any{
					"status": handlers.RestHealthStatusUp,
					"queue":  map[string]any{"pending": float64(0)},
				},
			},
		},
		{
			name: "Success/PostgresDegraded",
			jsonKeysMock: jsonKeysMock{
				response: &servicejsonkeys.StatusResponse{},
			},
			genaiMock: genaiMock{
				response: &servicegenai.StatusResponse{Queue: &servicegenai.QueueDepth{}},
			},
			skipPostgres: true,
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusDown},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:genai": map[string]any{
					"status": handlers.RestHealthStatusUp,
					"queue":  map[string]any{"pending": float64(0)},
				},
			},
		},
		{
			name: "Success/JSONKeysDegraded",
			jsonKeysMock: jsonKeysMock{
				err: errJSONKeys,
			},
			genaiMock: genaiMock{
				response: &servicegenai.StatusResponse{Queue: &servicegenai.QueueDepth{}},
			},
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusUp},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusDown,
				},
				"client:genai": map[string]any{
					"status": handlers.RestHealthStatusUp,
					"queue":  map[string]any{"pending": float64(0)},
				},
			},
		},
		{
			name: "Success/GenAIDegraded",
			jsonKeysMock: jsonKeysMock{
				response: &servicejsonkeys.StatusResponse{},
			},
			genaiMock: genaiMock{
				err: errGenAI,
			},
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusUp},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:genai": map[string]any{"status": handlers.RestHealthStatusDown},
			},
		},
		{
			name: "Success/GenAIQueueMissing",
			jsonKeysMock: jsonKeysMock{
				response: &servicejsonkeys.StatusResponse{},
			},
			genaiMock: genaiMock{
				response: &servicegenai.StatusResponse{},
			},
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusUp},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:genai": map[string]any{"status": handlers.RestHealthStatusDown},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			jsonKeysClient := servicejsonkeysmocks.NewMockBaseClient(t)
			jsonKeysClient.EXPECT().
				Status(mock.Anything, &servicejsonkeys.StatusRequest{}).
				Return(testCase.jsonKeysMock.response, testCase.jsonKeysMock.err).
				Once()

			genaiClient := servicegenaimocks.NewMockClient(t)
			genaiClient.EXPECT().
				Status(mock.Anything, &servicegenai.StatusRequest{}).
				Return(testCase.genaiMock.response, testCase.genaiMock.err).
				Once()

			handler := handlers.NewRestHealth(jsonKeysClient, genaiClient)

			requestContext := t.Context()

			if !testCase.skipPostgres {
				var err error

				requestContext, err = postgres.NewContext(requestContext, configtest.PostgresPreset)
				require.NoError(t, err)
			}

			requestCount := max(testCase.requestCount, 1)
			responses := make(chan *http.Response, requestCount)

			var requests sync.WaitGroup
			requests.Add(requestCount)

			for range requestCount {
				go func() {
					defer requests.Done()

					w := httptest.NewRecorder()
					r := httptest.NewRequestWithContext(
						requestContext,
						http.MethodGet,
						"/healthcheck",
						nil,
					)
					handler.ServeHTTP(w, r)

					responses <- w.Result()
				}()
			}

			requests.Wait()
			close(responses)

			for res := range responses {
				require.Equal(t, http.StatusOK, res.StatusCode)

				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonResponse any
				require.NoError(t, json.Unmarshal(data, &jsonResponse))
				require.Equal(t, testCase.expectResponse, jsonResponse)
			}

			jsonKeysClient.AssertExpectations(t)
			genaiClient.AssertExpectations(t)
		})
	}
}
