package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-narrative-engine/v2/internal/config"
	"github.com/a-novel/service-narrative-engine/v2/internal/config/env"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
)

func TestRestLimits(t *testing.T) {
	t.Parallel()

	otherUserID := uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	const (
		content = "/v1/instance/content"
		links   = "/v1/instance/links"
	)

	// documentWrite is one write a test case sends, as a user, to a route.
	type documentWrite struct {
		userID uuid.UUID
		path   string

		expectStatus int
	}

	// writes returns count writes of a user to a route, which all pass.
	writes := func(count int, userID uuid.UUID, path string) []documentWrite {
		sent := make([]documentWrite, count)
		for index := range sent {
			sent[index] = documentWrite{userID: userID, path: path, expectStatus: http.StatusOK}
		}

		return sent
	}

	// The default rate lets 3 writes through each minute.
	perMinute := env.RestRateLimitDocumentWriteDefault.Requests

	testCases := []struct {
		name string

		// rateLimitsDisabled turns the rates off, as a test stack does.
		rateLimitsDisabled bool
		writes             []documentWrite
	}{
		{
			name: "Success",

			writes: writes(perMinute, ownerID, content),
		},
		{
			// Each user writes on a budget of their own.
			name: "Success/PerUser",

			writes: append(writes(perMinute, ownerID, content), writes(1, otherUserID, content)...),
		},
		{
			// Each route that writes a document counts on a budget of its own.
			name: "Success/PerRoute",

			writes: append(writes(perMinute, ownerID, content), writes(1, ownerID, links)...),
		},
		{
			name: "Success/RateLimitsDisabled",

			rateLimitsDisabled: true,
			writes:             writes(perMinute+1, ownerID, content),
		},
		{
			name: "Error/OneWriteTooMany",

			writes: append(
				writes(perMinute, ownerID, content),
				documentWrite{userID: ownerID, path: content, expectStatus: http.StatusTooManyRequests},
			),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// The case sends its writes at once, well inside the minute the rate counts over.
			limit := handlers.NewRestLimits(
				httpf.ServerConfig{RateLimitsEnabled: !testCase.rateLimitsDisabled},
				config.Limits{RateLimitDocumentWrite: env.RestRateLimitDocumentWriteDefault},
			)

			answer := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

			router := chi.NewRouter()
			router.With(limit.DocumentWrite).Put(content, answer)
			router.With(limit.DocumentWrite).Put(links, answer)

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
