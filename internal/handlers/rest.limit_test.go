package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-narrative-engine/v2/internal/config"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
)

func TestRestLimits(t *testing.T) {
	t.Parallel()

	otherUserID := uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	// documentWrite is one write a test case sends, as a user, to a route.
	type documentWrite struct {
		userID uuid.UUID
		path   string

		expectStatus int
	}

	testCases := []struct {
		name string

		// rateLimitsDisabled turns the rates off, as a test stack does.
		rateLimitsDisabled bool
		writes             []documentWrite
	}{
		{
			name: "Success",

			writes: []documentWrite{
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusOK},
			},
		},
		{
			// Each user writes on a budget of their own.
			name: "Success/PerUser",

			writes: []documentWrite{
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusOK},
				{userID: otherUserID, path: "/v1/instance/content", expectStatus: http.StatusOK},
			},
		},
		{
			// Each route that writes a document counts on a budget of its own.
			name: "Success/PerRoute",

			writes: []documentWrite{
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/links", expectStatus: http.StatusOK},
			},
		},
		{
			name: "Success/RateLimitsDisabled",

			rateLimitsDisabled: true,
			writes: []documentWrite{
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusOK},
			},
		},
		{
			name: "Error/ThirdWrite",

			writes: []documentWrite{
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/instance/content", expectStatus: http.StatusTooManyRequests},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// The rate lets 2 writes through each minute, and the case sends its writes at once.
			limit := handlers.NewRestLimits(
				httpf.ServerConfig{RateLimitsEnabled: !testCase.rateLimitsDisabled},
				config.Limits{RateLimitDocumentWrite: lo.Must(httpf.ParseRate("2/1m"))},
			)

			answer := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

			router := chi.NewRouter()
			router.With(limit.DocumentWrite).Put("/v1/instance/content", answer)
			router.With(limit.DocumentWrite).Put("/v1/links", answer)

			for _, write := range testCase.writes {
				w := httptest.NewRecorder()
				request := httptest.NewRequestWithContext(t.Context(), http.MethodPut, write.path, nil)
				router.ServeHTTP(w, asUser(request, &write.userID))

				require.Equal(t, write.expectStatus, w.Code, write.path)

				// A refusal says when to try again.
				if write.expectStatus == http.StatusTooManyRequests {
					require.NotEmpty(t, w.Header().Get("Retry-After"))
				}
			}
		})
	}
}
