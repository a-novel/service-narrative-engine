package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
)

var (
	restOwnerID         = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	restProjectID       = uuid.MustParse("00000000-0000-0000-0000-000000000201")
	restIdeaVersionID   = uuid.MustParse("00000000-0000-0000-0000-000000000202")
	restStepValueID     = uuid.MustParse("00000000-0000-0000-0000-000000000401")
	restManuscriptID    = uuid.MustParse("00000000-0000-0000-0000-000000000501")
	restGenerationID    = uuid.MustParse("00000000-0000-0000-0000-000000000601")
	restCreatedAt       = time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
	restUpdatedAt       = restCreatedAt.Add(time.Minute)
	restStepValue       = json.RawMessage(`{"formerSchema":"intentionally opaque"}`)
	restManuscriptValue = json.RawMessage(
		`{"blocks":[{"type":"text","data":{"text":"The foghorn answered."},"metadata":{}}]}`,
	)
)

func executeRestHandler(
	t *testing.T,
	handler http.Handler,
	method string,
	pattern string,
	path string,
	body string,
	claims *serviceauthentication.Claims,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Method(method, pattern, handler)

	ctx := t.Context()
	if claims != nil {
		ctx = serviceauthentication.SetClaimsContext(ctx, claims)
	}

	request := httptest.NewRequestWithContext(
		ctx,
		method,
		path,
		bytes.NewBufferString(body),
	)
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}
