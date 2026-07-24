package daotest_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/dao/daotest"
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

// providerDeadline is short enough to keep the suite quick and long enough that a machine under
// load does not trip it before the request is even sent.
const providerDeadline = 250 * time.Millisecond

func TestProviderServer(t *testing.T) {
	t.Parallel()

	t.Run("Success/ReplaysAndRecords", func(t *testing.T) {
		t.Parallel()

		golden := daotest.Golden(t, "providerResponse.json")

		provider := daotest.NewProviderServer(t, daotest.ProviderResponse{
			Status: http.StatusCreated,
			Header: http.Header{"Content-Type": []string{"application/json"}},
			Body:   golden,
		})

		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			provider.URL()+"/generate?model=test",
			strings.NewReader(`{"prompt":"hi"}`),
		)
		require.NoError(t, err)

		response, err := lib.NewHTTPClient(lib.HTTPClientOptions{}).Do(request)
		require.NoError(t, err)

		body, err := io.ReadAll(response.Body)
		require.NoError(t, errors.Join(err, response.Body.Close()))

		require.Equal(t, http.StatusCreated, response.StatusCode)
		require.Equal(t, "application/json", response.Header.Get("Content-Type"))
		require.JSONEq(t, golden, string(body))

		recorded := provider.Requests()
		require.Len(t, recorded, 1)
		require.Equal(t, http.MethodPost, recorded[0].Method)
		require.Equal(t, "/generate", recorded[0].Path)
		require.Equal(t, "test", recorded[0].Query.Get("model"))
		require.JSONEq(t, `{"prompt":"hi"}`, string(recorded[0].Body))
	})

	t.Run("Error/HeldOpenPastTheDeadline", func(t *testing.T) {
		t.Parallel()

		provider := daotest.NewProviderServer(t, daotest.ProviderResponse{Hang: true})

		ctx, cancel := context.WithTimeout(t.Context(), providerDeadline)
		defer cancel()

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.URL(), nil)
		require.NoError(t, err)

		//nolint:bodyclose // The call never returns a response, so there is no body to close.
		_, err = lib.NewHTTPClient(lib.HTTPClientOptions{}).Do(request)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		require.Len(t, provider.Requests(), 1)
	})

	t.Run("Error/DroppedConnection", func(t *testing.T) {
		t.Parallel()

		provider := daotest.NewProviderServer(t, daotest.ProviderResponse{Drop: true})

		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, provider.URL(), nil)
		require.NoError(t, err)

		//nolint:bodyclose // The call never returns a response, so there is no body to close.
		_, err = lib.NewHTTPClient(lib.HTTPClientOptions{}).Do(request)
		require.Error(t, err)

		require.Len(t, provider.Requests(), 1)
	})
}
