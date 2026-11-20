package handlers_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestPluginModuleGet(t *testing.T) {
	t.Parallel()

	const (
		code     = `export default { mount() {} };`
		checksum = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
		etag     = `"` + checksum + `"`
	)

	target := "/v1/plugin/version/ui?plugin=scenes-composer&version=1.10.0"
	module := &core.PluginModule{Code: []byte(code), Checksum: checksum, PublishedAt: preciseAt}

	testCases := []struct {
		name string

		// module is the module the endpoint serves, the UI when empty.
		module      core.PluginModuleKind
		target      string
		ifNoneMatch string

		serviceMock *serviceMock[*core.PluginModule]

		expectStatus int
		expectBody   string
	}{
		{
			name: "Success",

			target: target,

			serviceMock: &serviceMock[*core.PluginModule]{resp: module},

			expectStatus: http.StatusOK,
			expectBody:   code,
		},
		{
			name: "Success/Migration",

			module: core.PluginModuleMigration,
			target: "/v1/plugin/version/migration?plugin=scenes-composer&version=1.10.0",

			serviceMock: &serviceMock[*core.PluginModule]{resp: module},

			expectStatus: http.StatusOK,
			expectBody:   code,
		},
		{
			name: "Success/NotModified",

			target:      target,
			ifNoneMatch: etag,

			serviceMock: &serviceMock[*core.PluginModule]{resp: module},

			expectStatus: http.StatusNotModified,
		},
		{
			name: "Success/OtherETag",

			// A cache that holds other bytes gets the module again.
			target:      target,
			ifNoneMatch: `"0000000000000000000000000000000000000000000000000000000000000000"`,

			serviceMock: &serviceMock[*core.PluginModule]{resp: module},

			expectStatus: http.StatusOK,
			expectBody:   code,
		},
		{
			name: "Error/MalformedRequest",

			// The decoder refuses a parameter the endpoint does not take.
			target: target + "&unknown=1",

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only pluginErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests").
			name: "Error/ModuleNotFound",

			target: target,

			serviceMock: &serviceMock[*core.PluginModule]{err: core.ErrPluginModuleNotFound},

			expectStatus: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			module := testCase.module
			if module == "" {
				module = core.PluginModuleUI
			}

			service := handlersmocks.NewMockRestPluginModuleGetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.PluginModuleGetRequest{
						Plugin: "scenes-composer", Version: "1.10.0", Module: module,
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			request := newRequest(t, http.MethodGet, testCase.target, nil)
			if testCase.ifNoneMatch != "" {
				request.Header.Set("If-None-Match", testCase.ifNoneMatch)
			}

			w := httptest.NewRecorder()
			handlers.NewRestPluginModuleGet(service, module, configtest.Logger).ServeHTTP(w, request)

			response := w.Result()
			require.Equal(t, testCase.expectStatus, response.StatusCode)

			body, err := io.ReadAll(response.Body)
			require.NoError(t, errors.Join(err, response.Body.Close()))

			if testCase.expectStatus >= http.StatusBadRequest {
				return
			}

			require.Equal(t, testCase.expectBody, string(body))
			require.Equal(t, etag, response.Header.Get("ETag"))
			require.Equal(t, "private, max-age=31536000, immutable", response.Header.Get("Cache-Control"))

			// A 304 leaves out the headers that describe the body.
			if testCase.expectStatus == http.StatusOK {
				require.Equal(t, "text/javascript; charset=utf-8", response.Header.Get("Content-Type"))
				require.Equal(t, "nosniff", response.Header.Get("X-Content-Type-Options"))
				require.Equal(t, "Sun, 03 Jan 2021 08:15:30 GMT", response.Header.Get("Last-Modified"))
			}
		})
	}
}
