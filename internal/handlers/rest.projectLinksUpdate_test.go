package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/v2/internal/handlers/mocks"
)

func TestRestProjectLinksUpdate(t *testing.T) {
	t.Parallel()

	otherInstanceID := uuid.MustParse("00000000-0000-0000-0000-000000000032")

	jsonLink := map[string]any{
		"source":       map[string]any{"id": instanceID.String(), "target": "scenes/a"},
		"destination":  map[string]any{"id": otherInstanceID.String(), "target": "cast/b"},
		"relationship": "therefore",
	}

	coreLink := core.Link{
		Source:       core.Identifier{ID: instanceID, Target: "scenes/a"},
		Destination:  core.Identifier{ID: otherInstanceID, Target: "cast/b"},
		Relationship: "therefore",
	}

	// newBody returns the body of a write of links read at updatedAt, a nil one omitting each.
	newBody := func(links, updatedAt any) map[string]any {
		body := map[string]any{"projectID": projectID.String()}

		if links != nil {
			body["links"] = links
		}

		if updatedAt != nil {
			body["updatedAt"] = updatedAt
		}

		return body
	}

	written := &serviceMock[*core.WrittenProjectLinks]{resp: &core.WrittenProjectLinks{UpdatedAt: preciseAt}}
	jsonWritten := map[string]any{"updatedAt": "2021-01-03T08:15:30.123456Z"}

	testCases := []struct {
		name string

		body any
		// anonymous sends the request for an anonymous session.
		anonymous bool

		// expectRequest is the request the service receives. A nil one means the handler must not call it.
		expectRequest *core.ProjectLinksUpdateRequest
		serviceMock   *serviceMock[*core.WrittenProjectLinks]

		expectStatus   int
		expectResponse any
	}{
		{
			// The request's updatedAt reaches the service with its microseconds.
			name: "Success",

			body: newBody([]any{jsonLink}, "2021-01-03T08:15:30.123456Z"),

			expectRequest: &core.ProjectLinksUpdateRequest{
				OwnerID: ownerID, ProjectID: projectID, Links: []core.Link{coreLink}, UpdatedAt: &preciseAt,
			},
			serviceMock: written,

			expectStatus:   http.StatusOK,
			expectResponse: jsonWritten,
		},
		{
			// A project that never wrote its links names no instant, and an empty list reaches the service as
			// one.
			name: "Success/FirstWrite",

			body: newBody([]any{}, nil),

			expectRequest: &core.ProjectLinksUpdateRequest{OwnerID: ownerID, ProjectID: projectID, Links: []core.Link{}},
			serviceMock:   written,

			expectStatus:   http.StatusOK,
			expectResponse: jsonWritten,
		},
		{
			// A list the body omits reaches the service as none, which it refuses.
			name: "Error/NoLinks",

			body: newBody(nil, "2021-01-03T08:15:30.123456Z"),

			expectRequest: &core.ProjectLinksUpdateRequest{OwnerID: ownerID, ProjectID: projectID, UpdatedAt: &preciseAt},
			serviceMock:   &serviceMock[*core.WrittenProjectLinks]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/Anonymous",

			body:      newBody([]any{jsonLink}, nil),
			anonymous: true,

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/MalformedRequest",

			body: "not an object",

			expectStatus: http.StatusBadRequest,
		},
		{
			// An ID that is no UUID fails the decoding.
			name: "Error/MalformedID",

			body: newBody([]any{map[string]any{"source": map[string]any{"id": "nope", "target": "a"}}}, nil),

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only instanceErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests").
			name: "Error/Stale",

			body: newBody([]any{}, nil),

			expectRequest: &core.ProjectLinksUpdateRequest{OwnerID: ownerID, ProjectID: projectID, Links: []core.Link{}},
			serviceMock:   &serviceMock[*core.WrittenProjectLinks]{err: core.ErrProjectLinksStale},

			expectStatus: http.StatusConflict,
		},
		{
			name: "Error/ProjectBudget",

			body: newBody([]any{}, nil),

			expectRequest: &core.ProjectLinksUpdateRequest{OwnerID: ownerID, ProjectID: projectID, Links: []core.Link{}},
			serviceMock:   &serviceMock[*core.WrittenProjectLinks]{err: core.ErrProjectBudget},

			expectStatus:   http.StatusConflict,
			expectResponse: limitProblem("projectBytes"),
		},
		{
			// Links over a cap name their field and the rule it broke.
			name: "Error/InvalidField",

			body: newBody([]any{}, nil),

			expectRequest: &core.ProjectLinksUpdateRequest{OwnerID: ownerID, ProjectID: projectID, Links: []core.Link{}},
			serviceMock: &serviceMock[*core.WrittenProjectLinks]{
				err: &core.FieldError{Path: "links[3].relationship", Rule: "maxbytes"},
			},

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: invalidFieldsProblem(map[string]any{"links[3].relationship": "maxbytes"}),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectLinksUpdateService(t)

			if testCase.expectRequest != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.expectRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			request := newRequest(t, http.MethodPut, "/v1/links", testCase.body)
			if testCase.anonymous {
				request = asUser(request, nil)
			}

			w := httptest.NewRecorder()
			handlers.NewRestProjectLinksUpdate(service, configtest.Logger).ServeHTTP(w, request)

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
