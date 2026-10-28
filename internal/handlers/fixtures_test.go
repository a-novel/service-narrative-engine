package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

var (
	ownerID   = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	projectID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sceneID   = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	layerID   = uuid.MustParse("00000000-0000-0000-0000-000000000021")

	createdAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt = time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)

	coreProject = &core.Project{
		ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Premise: "Marie attend.", Language: "fr",
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
	jsonProject = map[string]any{
		"id": projectID.String(), "title": "Les Heures claires", "premise": "Marie attend.", "language": "fr",
		"createdAt": "2021-01-01T00:00:00Z", "updatedAt": "2021-01-02T00:00:00Z",
	}

	coreScene = &core.Scene{
		ID: sceneID, ProjectID: projectID, Position: 1, Title: "Ouverture", Color: "#e0457b",
		Description: "Marie attend.", Prompt: "Lent.", UpdatedAt: updatedAt,
	}
	jsonScene = map[string]any{
		"id": sceneID.String(), "projectID": projectID.String(), "position": 1.0, "title": "Ouverture",
		"color": "#e0457b", "description": "Marie attend.", "prompt": "Lent.", "updatedAt": "2021-01-02T00:00:00Z",
	}

	coreLayer = &core.Layer{
		ID: layerID, ProjectID: projectID, Position: 1, Kind: core.LayerKindNotes, Name: "Characters",
		Description: "Who appears", UpdatedAt: updatedAt,
	}
	jsonLayer = map[string]any{
		"id": layerID.String(), "projectID": projectID.String(), "position": 1.0, "kind": "notes",
		"name": "Characters", "description": "Who appears", "updatedAt": "2021-01-02T00:00:00Z",
	}

	coreCell = &core.Cell{
		ProjectID: projectID, SceneID: sceneID, LayerID: layerID, Content: json.RawMessage(`{"text":"Marie"}`),
		UpdatedAt: updatedAt,
	}
	jsonCell = map[string]any{
		"projectID": projectID.String(), "sceneID": sceneID.String(), "layerID": layerID.String(),
		"content":   map[string]any{"text": "Marie"},
		"updatedAt": "2021-01-02T00:00:00Z",
	}
)

// withComposer returns the JSON of a new project carrying the layers it starts with.
func withComposer(project map[string]any, layers []any) map[string]any {
	composer := maps.Clone(project)
	composer["layers"] = layers

	return composer
}

// asUser returns the request as the auth middleware hands it over: carrying the claims of userID,
// or of an anonymous session when userID is nil.
func asUser(request *http.Request, userID *uuid.UUID) *http.Request {
	claims := &serviceauthentication.Claims{UserID: userID}

	return request.WithContext(serviceauthentication.SetClaimsContext(request.Context(), claims))
}

// newRequest returns a request of the project owner, carrying body as JSON unless it is nil.
func newRequest(t *testing.T, method, target string, body any) *http.Request {
	t.Helper()

	var data []byte

	if body != nil {
		var err error

		data, err = json.Marshal(body)
		require.NoError(t, err)
	}

	return asUser(httptest.NewRequestWithContext(t.Context(), method, target, bytes.NewReader(data)), &ownerID)
}

// requireResponse checks the status of a handler's response, and its JSON body when one is
// expected.
func requireResponse(t *testing.T, response *http.Response, expectStatus int, expectBody any) {
	t.Helper()

	require.Equal(t, expectStatus, response.StatusCode)

	if expectBody == nil {
		return
	}

	data, err := io.ReadAll(response.Body)
	require.NoError(t, errors.Join(err, response.Body.Close()))

	var body any
	require.NoError(t, json.Unmarshal(data, &body))
	require.Equal(t, expectBody, body)
}
