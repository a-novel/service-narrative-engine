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

func TestRestInstanceLinksUpdate(t *testing.T) {
	t.Parallel()

	otherInstanceID := uuid.MustParse("00000000-0000-0000-0000-000000000032")

	jsonLink := map[string]any{
		"source":       "scenes/a",
		"destination":  map[string]any{"id": otherInstanceID.String(), "target": "cast/b"},
		"relationship": "therefore",
	}

	coreLink := core.Link{
		Source:       "scenes/a",
		Destination:  core.Identifier{ID: otherInstanceID, Target: "cast/b"},
		Relationship: "therefore",
	}

	// newBody returns the body of a write of the links of the test instance read at updatedAt, a nil one
	// omitting each.
	newBody := func(links, updatedAt any) map[string]any {
		body := map[string]any{"projectID": projectID.String(), "id": instanceID.String()}

		if links != nil {
			body["links"] = links
		}

		if updatedAt != nil {
			body["updatedAt"] = updatedAt
		}

		return body
	}

	// emptied is the request of a write that names no instant and clears the links.
	emptied := &core.InstanceLinksUpdateRequest{
		OwnerID: ownerID, ProjectID: projectID, ID: instanceID, Links: []core.Link{},
	}

	written := &serviceMock[*core.WrittenInstanceLinks]{resp: &core.WrittenInstanceLinks{UpdatedAt: preciseAt}}
	jsonWritten := map[string]any{"updatedAt": "2021-01-03T08:15:30.123456Z"}

	testCases := []struct {
		name string

		body any
		// anonymous sends the request for an anonymous session.
		anonymous bool

		// expectRequest is the request the service receives. A nil one means the handler must not call it.
		expectRequest *core.InstanceLinksUpdateRequest
		serviceMock   *serviceMock[*core.WrittenInstanceLinks]

		expectStatus   int
		expectResponse any
	}{
		{
			// The request's updatedAt reaches the service with its microseconds.
			name: "Success",

			body: newBody([]any{jsonLink}, "2021-01-03T08:15:30.123456Z"),

			expectRequest: &core.InstanceLinksUpdateRequest{
				OwnerID: ownerID, ProjectID: projectID, ID: instanceID, Links: []core.Link{coreLink},
				UpdatedAt: &preciseAt,
			},
			serviceMock: written,

			expectStatus:   http.StatusOK,
			expectResponse: jsonWritten,
		},
		{
			// An instance that never wrote its links names no instant, and an empty list reaches the service as
			// one.
			name: "Success/FirstWrite",

			body: newBody([]any{}, nil),

			expectRequest: emptied,
			serviceMock:   written,

			expectStatus:   http.StatusOK,
			expectResponse: jsonWritten,
		},
		{
			// A list the body omits reaches the service as none, which it refuses.
			name: "Error/NoLinks",

			body: newBody(nil, "2021-01-03T08:15:30.123456Z"),

			expectRequest: &core.InstanceLinksUpdateRequest{
				OwnerID: ownerID, ProjectID: projectID, ID: instanceID, UpdatedAt: &preciseAt,
			},
			serviceMock: &serviceMock[*core.WrittenInstanceLinks]{err: core.ErrInvalidRequest},

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

			body: newBody([]any{map[string]any{"destination": map[string]any{"id": "nope", "target": "a"}}}, nil),

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only instanceErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests").
			name: "Error/Stale",

			body: newBody([]any{}, nil),

			expectRequest: emptied,
			serviceMock:   &serviceMock[*core.WrittenInstanceLinks]{err: core.ErrInstanceLinksStale},

			expectStatus: http.StatusConflict,
		},
		{
			name: "Error/ProjectBudget",

			body: newBody([]any{}, nil),

			expectRequest: emptied,
			serviceMock:   &serviceMock[*core.WrittenInstanceLinks]{err: core.ErrProjectBudget},

			expectStatus:   http.StatusConflict,
			expectResponse: limitProblem("projectBytes"),
		},
		{
			// Links over a cap name their field and the rule it broke.
			name: "Error/InvalidField",

			body: newBody([]any{}, nil),

			expectRequest: emptied,
			serviceMock: &serviceMock[*core.WrittenInstanceLinks]{
				err: &core.FieldError{Path: "links[3].relationship", Rule: "maxbytes"},
			},

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: invalidFieldsProblem(map[string]any{"links[3].relationship": "maxbytes"}),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestInstanceLinksUpdateService(t)

			if testCase.expectRequest != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.expectRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			request := newRequest(t, http.MethodPut, "/v1/instance/links", testCase.body)
			if testCase.anonymous {
				request = asUser(request, nil)
			}

			w := httptest.NewRecorder()
			handlers.NewRestInstanceLinksUpdate(service, configtest.Logger).ServeHTTP(w, request)

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
