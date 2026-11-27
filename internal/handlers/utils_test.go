package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/postgres"
	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
)

// serviceMock is what the service mock of a handler test case returns. A nil one means the handler
// must not call the service.
type serviceMock[Resp any] struct {
	resp Resp
	err  error
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

// probeSetup selects the state of the PostgreSQL connection a health probe meets.
type probeSetup int

const (
	// probeHealthy gives the probe an open connection.
	probeHealthy probeSetup = iota
	// probeWithoutPostgres leaves PostgreSQL out of the context.
	probeWithoutPostgres
	// probeClosedPostgres gives the probe a closed connection pool.
	probeClosedPostgres
)

// newProbeContext returns the context a health probe runs with under the given setup. Each call
// opens a pool of its own, so closing it leaves the other tests' connections alone.
func newProbeContext(t *testing.T, setup probeSetup) context.Context {
	t.Helper()

	ctx := t.Context()
	if setup == probeWithoutPostgres {
		return ctx
	}

	ctx, err := postgres.NewContext(ctx, postgrespresets.NewDefault(configtest.PostgresPreset.Options()...))
	require.NoError(t, err)

	pg, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	db, ok := pg.(*bun.DB)
	require.True(t, ok)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	if setup == probeClosedPostgres {
		require.NoError(t, db.Close())
	}

	return ctx
}

// limitProblem is the body of a refusal for a storage limit, whose limit tag names the limit reached.
func limitProblem(limit string) any {
	return map[string]any{
		"type": "about:blank", "title": "Conflict", "status": 409.0, "tags": map[string]any{"limit": limit},
	}
}

// invalidFieldsProblem is the body of a refusal for invalid request fields, each named by its path.
func invalidFieldsProblem(fields map[string]any) any {
	return map[string]any{
		"type": "about:blank", "title": "Unprocessable Entity", "status": 422.0,
		"tags": map[string]any{"invalidFields": fields},
	}
}
