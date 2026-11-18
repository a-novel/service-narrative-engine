package handlers_test

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestBeatsDelete(t *testing.T) {
	t.Parallel()

	// The request deletes a second beat too, so its id parameter repeats.
	otherBeat := *coreBeat
	otherBeat.ID, otherBeat.Position = uuid.MustParse("00000000-0000-0000-0000-000000000012"), 2

	otherJSON := maps.Clone(jsonBeat)
	otherJSON["id"], otherJSON["position"] = otherBeat.ID.String(), 2.0

	target := "/v1/beats?projectID=" + projectID.String() + "&id=" + beatID.String() + "&id=" + otherBeat.ID.String()

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodDelete, target, nil)
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[[]*core.Beat]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Beat]{resp: []*core.Beat{coreBeat, &otherBeat}},

			expectStatus:   http.StatusOK,
			expectResponse: []any{jsonBeat, otherJSON},
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

				return newRequest(t, http.MethodDelete, "/v1/beats?projectID="+projectID.String()+"&id=not-a-uuid", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only composerErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests"), whether or not the route produces it.
			name: "Error/ProjectFull",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Beat]{err: core.ErrProjectFull},

			expectStatus: http.StatusConflict,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestBeatsDeleteService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.BeatsDeleteRequest{
						OwnerID: ownerID, ProjectID: projectID, IDs: []uuid.UUID{beatID, otherBeat.ID},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestBeatsDelete(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
