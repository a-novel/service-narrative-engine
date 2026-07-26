package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"
	servicejobsmocks "github.com/a-novel/service-jobs/pkg/go/mocks"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
	servicejsonkeysmocks "github.com/a-novel/service-json-keys/v2/pkg/go/mocks"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

func TestRestHealth(t *testing.T) {
	t.Parallel()

	errJSONKeys := errors.New("json keys unavailable")
	errJobs := errors.New("jobs unavailable")

	type jsonKeysMock struct {
		response *servicejsonkeys.StatusResponse
		err      error
	}

	type jobsMock struct {
		response *servicejobs.StatusResponse
		err      error
	}

	testCases := []struct {
		name string

		jsonKeysMock jsonKeysMock
		jobsMock     jobsMock
		skipPostgres bool
		requestCount int

		expectResponse any
	}{
		{
			name: "Success/CachedBurst",
			jsonKeysMock: jsonKeysMock{
				response: &servicejsonkeys.StatusResponse{},
			},
			jobsMock: jobsMock{
				response: &servicejobs.StatusResponse{
					Queue: &servicejobs.QueueDepth{
						Pending:          3,
						OldestPendingAge: durationpb.New(90 * time.Second),
					},
				},
			},
			requestCount: 8,
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusUp},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:jobs": map[string]any{
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
			jobsMock: jobsMock{
				response: &servicejobs.StatusResponse{
					Queue: &servicejobs.QueueDepth{},
				},
			},
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusUp},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:jobs": map[string]any{
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
			jobsMock: jobsMock{
				response: &servicejobs.StatusResponse{Queue: &servicejobs.QueueDepth{}},
			},
			skipPostgres: true,
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusDown},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:jobs": map[string]any{
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
			jobsMock: jobsMock{
				response: &servicejobs.StatusResponse{Queue: &servicejobs.QueueDepth{}},
			},
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusUp},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusDown,
				},
				"client:jobs": map[string]any{
					"status": handlers.RestHealthStatusUp,
					"queue":  map[string]any{"pending": float64(0)},
				},
			},
		},
		{
			name: "Success/JobsDegraded",
			jsonKeysMock: jsonKeysMock{
				response: &servicejsonkeys.StatusResponse{},
			},
			jobsMock: jobsMock{
				err: errJobs,
			},
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusUp},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:jobs": map[string]any{"status": handlers.RestHealthStatusDown},
			},
		},
		{
			name: "Success/JobsQueueMissing",
			jsonKeysMock: jsonKeysMock{
				response: &servicejsonkeys.StatusResponse{},
			},
			jobsMock: jobsMock{
				response: &servicejobs.StatusResponse{},
			},
			expectResponse: map[string]any{
				"client:postgres": map[string]any{"status": handlers.RestHealthStatusUp},
				"client:json-keys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:jobs": map[string]any{"status": handlers.RestHealthStatusDown},
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

			jobsClient := servicejobsmocks.NewMockClient(t)
			jobsClient.EXPECT().
				Status(mock.Anything, &servicejobs.StatusRequest{}).
				Return(testCase.jobsMock.response, testCase.jobsMock.err).
				Once()

			handler := handlers.NewRestHealth(jsonKeysClient, jobsClient)

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
			jobsClient.AssertExpectations(t)
		})
	}
}
