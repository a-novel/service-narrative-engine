package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

func TestRestLimit(t *testing.T) {
	t.Parallel()

	type request struct {
		path    string
		visitor string

		expectStatus int
	}

	testCases := []struct {
		name string

		requests []request
	}{
		{
			name: "Success",

			requests: []request{
				{path: "/items", visitor: "192.0.2.1", expectStatus: http.StatusOK},
				{path: "/items", visitor: "192.0.2.1", expectStatus: http.StatusOK},
			},
		},
		{
			name: "Success/OtherVisitor",

			requests: []request{
				{path: "/items", visitor: "192.0.2.1", expectStatus: http.StatusOK},
				{path: "/items", visitor: "192.0.2.1", expectStatus: http.StatusOK},
				{path: "/items", visitor: "192.0.2.2", expectStatus: http.StatusOK},
			},
		},
		{
			name: "Success/OtherRoute",

			requests: []request{
				{path: "/items", visitor: "192.0.2.1", expectStatus: http.StatusOK},
				{path: "/items", visitor: "192.0.2.1", expectStatus: http.StatusOK},
				{path: "/items/1", visitor: "192.0.2.1", expectStatus: http.StatusOK},
			},
		},
		{
			name: "Error/LimitReached",

			requests: []request{
				{path: "/items", visitor: "192.0.2.1", expectStatus: http.StatusOK},
				{path: "/items", visitor: "192.0.2.1", expectStatus: http.StatusOK},
				{path: "/items", visitor: "192.0.2.1", expectStatus: http.StatusTooManyRequests},
			},
		},
		{
			// Path parameters must not open a fresh budget per value.
			name: "Error/LimitReached/PathParameters",

			requests: []request{
				{path: "/items/1", visitor: "192.0.2.1", expectStatus: http.StatusOK},
				{path: "/items/2", visitor: "192.0.2.1", expectStatus: http.StatusOK},
				{path: "/items/3", visitor: "192.0.2.1", expectStatus: http.StatusTooManyRequests},
			},
		},
		{
			// One IPv6 client controls its whole /64.
			name: "Error/LimitReached/IPv6Prefix",

			requests: []request{
				{path: "/items", visitor: "2001:db8::1", expectStatus: http.StatusOK},
				{path: "/items", visitor: "2001:db8::2", expectStatus: http.StatusOK},
				{path: "/items", visitor: "2001:db8::3", expectStatus: http.StatusTooManyRequests},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

			router := chi.NewRouter()
			router.Use(middleware.ClientIPFromHeader("X-Client-IP"))
			router.Group(func(api chi.Router) {
				api.Use(handlers.RestLimit(2, time.Minute))
				api.Get("/items", ok)
				api.Get("/items/{id}", ok)
			})

			for _, request := range testCase.requests {
				httpRequest := httptest.NewRequestWithContext(t.Context(), http.MethodGet, request.path, nil)
				httpRequest.Header.Set("X-Client-Ip", request.visitor)

				response := httptest.NewRecorder()
				router.ServeHTTP(response, httpRequest)

				require.Equal(t, request.expectStatus, response.Code, request.path)

				if request.expectStatus == http.StatusTooManyRequests {
					require.Equal(t, "60", response.Header().Get("Retry-After"))
				}
			}
		})
	}
}
