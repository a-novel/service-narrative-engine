package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/config/env"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

func TestRestLimits(t *testing.T) {
	t.Parallel()

	otherUserID := uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	// limitedRead is one read a test case sends, as a user, to a route.
	type limitedRead struct {
		userID uuid.UUID
		path   string

		expectStatus int
	}

	testCases := []struct {
		name string

		reads []limitedRead
	}{
		{
			name: "Success",

			reads: []limitedRead{
				{userID: ownerID, path: "/v1/generation", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/generation", expectStatus: http.StatusOK},
			},
		},
		{
			// Each user reads on a budget of their own.
			name: "Success/PerUser",

			reads: []limitedRead{
				{userID: ownerID, path: "/v1/generation", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/generation", expectStatus: http.StatusOK},
				{userID: otherUserID, path: "/v1/generation", expectStatus: http.StatusOK},
			},
		},
		{
			// A read and a stream each count on a budget of their own.
			name: "Success/PerRoute",

			reads: []limitedRead{
				{userID: ownerID, path: "/v1/generation", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/generation", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/generation/stream", expectStatus: http.StatusOK},
			},
		},
		{
			name: "Error/ThirdRead",

			reads: []limitedRead{
				{userID: ownerID, path: "/v1/generation", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/generation", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/generation", expectStatus: http.StatusTooManyRequests},
			},
		},
		{
			name: "Error/ThirdStream",

			reads: []limitedRead{
				{userID: ownerID, path: "/v1/generation/stream", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/generation/stream", expectStatus: http.StatusOK},
				{userID: ownerID, path: "/v1/generation/stream", expectStatus: http.StatusTooManyRequests},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// The default rate lets 2 reads through each second, and the case sends its reads at once.
			limit := handlers.NewRestLimits(httpf.ServerConfig{RateLimitsEnabled: true}, config.Limits{
				MaxStreamsPerCaller: env.RestMaxStreamsPerCallerDefault,
				GenerationRate:      env.RestRateLimitGenerationDefault,
				GenerationReadRate:  env.RestRateLimitGenerationReadDefault,
			})

			answer := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

			router := chi.NewRouter()
			router.With(limit.GenerationRead).Get("/v1/generation", answer)
			router.With(limit.GenerationStream).Get("/v1/generation/stream", answer)

			for _, read := range testCase.reads {
				w := httptest.NewRecorder()
				request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, read.path, nil)
				router.ServeHTTP(w, asUser(request, &read.userID))

				require.Equal(t, read.expectStatus, w.Code, read.path)
			}
		})
	}
}
