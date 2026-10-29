package handlers_test

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

// manuscriptGolden is the export of a French project with narration, dialogue with tones, and an empty
// scene. The JS client's tests expect the same document from a running service.
//
//go:embed testdata/manuscript.golden.json
var manuscriptGolden []byte

func TestRestManuscriptExport(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	// The handler answers the golden document, so its answer follows the format v1 schema too.
	require.NoError(t, schemas.Manuscript.Validate(manuscriptGolden))

	var jsonManuscript any
	require.NoError(t, json.Unmarshal(manuscriptGolden, &jsonManuscript))

	coreManuscript := &core.Manuscript{
		Title:    "Les Heures claires",
		Language: "fr",
		Scenes: []*core.ManuscriptScene{
			{Title: "Ouverture", Blocks: json.RawMessage(`[
				{"kind":"narration","text":"La pluie battait les vitres. Marie guettait la rue depuis l’aube."},
				{"kind":"dialogue","speaker":"Marie","parts":[
					{"tone":"à voix basse"},{"text":"Tu es venu."},
					{"tone":"plus fort, sans se retourner"},{"text":"Tu avais promis d’écrire."}
				]},
				{"kind":"dialogue","speaker":"Paul","parts":[{"text":"Je suis là, maintenant."}]}
			]`)},
			{Title: "Interlude", Blocks: json.RawMessage(`[]`)},
			{Title: "Dénouement", Blocks: json.RawMessage(
				`[{"kind":"narration","text":"Au matin, la pluie avait cessé.\n\nPaul était parti."}]`,
			)},
		},
	}

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodGet, "/v1/manuscript?projectID="+projectID.String()+"&layerID="+layerID.String(), nil)
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[*core.Manuscript]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.Manuscript]{resp: coreManuscript},

			expectStatus:   http.StatusOK,
			expectResponse: jsonManuscript,
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

				return newRequest(t, http.MethodGet, "/v1/manuscript?projectID="+projectID.String()+"&layerID=not-a-uuid", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.Manuscript]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/LayerNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.Manuscript]{err: core.ErrLayerNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidRequest",

			request: validRequest,

			serviceMock: &serviceMock[*core.Manuscript]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[*core.Manuscript]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestManuscriptExportService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ManuscriptExportRequest{OwnerID: ownerID, ProjectID: projectID, LayerID: layerID}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestManuscriptExport(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
