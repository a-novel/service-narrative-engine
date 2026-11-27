package handlers_test

import (
	"encoding/json"
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

func TestRestInstanceLinksList(t *testing.T) {
	t.Parallel()

	otherInstanceID := uuid.MustParse("00000000-0000-0000-0000-000000000032")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodGet, "/v1/links?projectID="+projectID.String()+"&limit=2&offset=1", nil)
	}

	// paged is the page [validRequest] asks for.
	paged := &core.InstanceLinksListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 2, Offset: 1}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		// serviceRequest is the request the service receives, [paged] when nil.
		serviceRequest *core.InstanceLinksListRequest
		serviceMock    *serviceMock[*core.InstanceLinksPage]

		expectStatus   int
		expectResponse any
	}{
		{
			// The first instance wrote its links, and its updatedAt keeps its microseconds. The second never did:
			// it answers an empty list and a null updatedAt.
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceLinksPage]{resp: &core.InstanceLinksPage{
				Instances: []*core.InstanceLinks{
					{
						InstanceID: instanceID, Links: json.RawMessage(`[{"relationship": "therefore"}]`),
						UpdatedAt: &preciseAt,
					},
					{InstanceID: otherInstanceID, Links: json.RawMessage(`[]`)},
				},
				Total: 3,
			}},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"instances": []any{
					map[string]any{
						"instanceID": instanceID.String(),
						"links":      []any{map[string]any{"relationship": "therefore"}},
						"updatedAt":  "2021-01-03T08:15:30.123456Z",
					},
					map[string]any{"instanceID": otherInstanceID.String(), "links": []any{}, "updatedAt": nil},
				},
				"total": 3.0,
			},
		},
		{
			// A request that names no page leaves the limit and the offset to the service.
			name: "Success/NoPage",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodGet, "/v1/links?projectID="+projectID.String(), nil)
			},

			serviceRequest: &core.InstanceLinksListRequest{OwnerID: ownerID, ProjectID: projectID},
			serviceMock: &serviceMock[*core.InstanceLinksPage]{resp: &core.InstanceLinksPage{
				Instances: []*core.InstanceLinks{},
			}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"instances": []any{}, "total": 0.0},
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
			name: "Error/MalformedID",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodGet, "/v1/links?projectID=nope", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/MalformedLimit",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodGet, "/v1/links?projectID="+projectID.String()+"&limit=many", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only instanceErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests").
			name: "Error/InstanceNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceLinksPage]{err: core.ErrInstanceNotFound},

			expectStatus: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestInstanceLinksListService(t)

			if testCase.serviceMock != nil {
				serviceRequest := testCase.serviceRequest
				if serviceRequest == nil {
					serviceRequest = paged
				}

				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestInstanceLinksList(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
