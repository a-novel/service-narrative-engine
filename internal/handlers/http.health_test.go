package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestHealth(t *testing.T) {
	t.Parallel()

	type healthApiJsonKeysMock struct {
		res *servicejsonkeys.StatusResponse
		err error
	}

	testCases := []struct {
		name string

		request *http.Request

		healthApiJsonKeysMock *healthApiJsonKeysMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil),

			healthApiJsonKeysMock: &healthApiJsonKeysMock{
				res: new(servicejsonkeys.StatusResponse),
			},

			expectResponse: map[string]any{
				"client:postgres": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"api:jsonKeys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil),

			healthApiJsonKeysMock: &healthApiJsonKeysMock{
				err: errors.New("error json keys"),
			},

			expectResponse: map[string]any{
				"client:postgres": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"api:jsonKeys": map[string]any{
					"status": handlers.RestHealthStatusDown,
					"err":    "error json keys",
				},
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			healthApiJsonKeys := handlersmocks.NewMockRestHealthApiJsonKeys(t)

			if testCase.healthApiJsonKeysMock != nil {
				healthApiJsonKeys.EXPECT().
					Status(mock.Anything, new(servicejsonkeys.StatusRequest)).
					Return(testCase.healthApiJsonKeysMock.res, testCase.healthApiJsonKeysMock.err)
			}

			handler := handlers.NewHealth(healthApiJsonKeys)
			w := httptest.NewRecorder()

			rCtx := testCase.request.Context()
			rCtx, err := postgres.NewContext(rCtx, config.PostgresPresetTest)
			require.NoError(t, err)

			handler.ServeHTTP(w, testCase.request.WithContext(rCtx))

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonRes any
				require.NoError(t, json.Unmarshal(data, &jsonRes))
				require.Equal(t, testCase.expectResponse, jsonRes)
			}
		})
	}
}
