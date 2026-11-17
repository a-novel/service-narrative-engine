package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestProjectUpdate(t *testing.T) {
	t.Parallel()

	tags := []string{"noir", "huis clos"}
	genres := []string{"thriller"}

	fields := func() map[string]any {
		return map[string]any{
			"id":       projectID,
			"title":    "Les Heures claires",
			"premise":  "Marie attend.",
			"language": "fr",
			"tags":     tags,
			"genres":   genres,
		}
	}

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPut, "/v1/project", fields())
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		// expectTags is the list the service receives: empty for a request that clears the tags, nil for one that
		// omits them, which the service refuses.
		expectTags []string
		// expectGenres follows the same rule as expectTags.
		expectGenres []string
		serviceMock  *serviceMock[*core.Project]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			expectTags:   tags,
			expectGenres: genres,
			serviceMock:  &serviceMock[*core.Project]{resp: coreProject},

			expectStatus:   http.StatusOK,
			expectResponse: jsonProject,
		},
		{
			name: "Success/ClearTags",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				body := fields()
				body["tags"] = []string{}

				return newRequest(t, http.MethodPut, "/v1/project", body)
			},

			expectTags:   []string{},
			expectGenres: genres,
			serviceMock:  &serviceMock[*core.Project]{resp: coreProject},

			expectStatus:   http.StatusOK,
			expectResponse: jsonProject,
		},
		{
			name: "Error/TagsOmitted",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				body := fields()
				delete(body, "tags")

				return newRequest(t, http.MethodPut, "/v1/project", body)
			},

			expectGenres: genres,
			serviceMock:  &serviceMock[*core.Project]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Success/ClearGenres",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				body := fields()
				body["genres"] = []string{}

				return newRequest(t, http.MethodPut, "/v1/project", body)
			},

			expectTags:   tags,
			expectGenres: []string{},
			serviceMock:  &serviceMock[*core.Project]{resp: coreProject},

			expectStatus:   http.StatusOK,
			expectResponse: jsonProject,
		},
		{
			name: "Error/GenresOmitted",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				body := fields()
				delete(body, "genres")

				return newRequest(t, http.MethodPut, "/v1/project", body)
			},

			expectTags:  tags,
			serviceMock: &serviceMock[*core.Project]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
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

				return newRequest(t, http.MethodPut, "/v1/project", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only composerErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests"), whether or not the route produces it.
			name: "Error/ProjectsFull",

			request: validRequest,

			expectTags:   tags,
			expectGenres: genres,
			serviceMock:  &serviceMock[*core.Project]{err: core.ErrProjectsFull},

			expectStatus:   http.StatusConflict,
			expectResponse: limitProblem("projects"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectUpdateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ProjectUpdateRequest{
						OwnerID:  ownerID,
						ID:       projectID,
						Title:    "Les Heures claires",
						Premise:  "Marie attend.",
						Language: "fr",
						Tags:     testCase.expectTags,
						Genres:   testCase.expectGenres,
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestProjectUpdate(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
