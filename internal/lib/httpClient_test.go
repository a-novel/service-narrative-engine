package lib_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

const (
	// slowHeaderDelay outlasts any header timeout a hardening pass would plausibly introduce while
	// still keeping the test quick.
	slowHeaderDelay = 250 * time.Millisecond

	// traceID and the traceparent carrying it stand in for an inbound trace this service continues.
	traceID            = "0102030405060708090a0b0c0d0e0f10"
	inboundTraceHeader = "00-" + traceID + "-0102030405060708-01"
)

func TestNewTransport(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		options lib.HTTPClientOptions
	}{
		{
			name: "Success",

			options: lib.HTTPClientOptions{MaxIdleConns: 100, MaxIdleConnsPerHost: 4},
		},
		{
			name: "Success/Zeroed",

			options: lib.HTTPClientOptions{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			transport := lib.NewTransport(testCase.options)

			require.Equal(t, testCase.options.MaxIdleConns, transport.MaxIdleConns)
			require.Equal(t, testCase.options.MaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)

			// A live guard on the decision written out in NewTransport: a hardening change that
			// sets this would break only long generations, which is to say only in production.
			require.Zero(t, transport.ResponseHeaderTimeout)
		})
	}
}

func TestNewHTTPClient(t *testing.T) {
	t.Parallel()

	t.Run("Success/NoOverallTimeout", func(t *testing.T) {
		t.Parallel()

		// The companion guard to the transport's zero header timeout: this one caps the whole
		// exchange, so a non-zero value would truncate a long generation mid-body.
		require.Zero(t, lib.NewHTTPClient(lib.HTTPClientOptions{}).Timeout)
	})

	t.Run("Success/ReusesConnections", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		client := lib.NewHTTPClient(lib.HTTPClientOptions{MaxIdleConns: 100, MaxIdleConnsPerHost: 4})

		var reused []bool

		for range 3 {
			ctx := httptrace.WithClientTrace(t.Context(), &httptrace.ClientTrace{
				GotConn: func(info httptrace.GotConnInfo) {
					reused = append(reused, info.Reused)
				},
			})

			request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			require.NoError(t, err)

			response, err := client.Do(request)
			require.NoError(t, err)

			// Draining and closing is what returns the connection to the idle pool. Skipping either
			// makes every request below open a fresh one, so this is part of the assertion.
			_, err = io.Copy(io.Discard, response.Body)
			require.NoError(t, errors.Join(err, response.Body.Close()))
		}

		require.Equal(t, []bool{false, true, true}, reused)
	})

	t.Run("Success/SlowResponseHeaders", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(slowHeaderDelay)
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		client := lib.NewHTTPClient(lib.HTTPClientOptions{})

		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		response, err := client.Do(request)
		require.NoError(t, err)

		defer func() { require.NoError(t, response.Body.Close()) }()

		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("Success/PropagatesTraceContext", func(t *testing.T) {
		t.Parallel()

		// Buffered so the handler never blocks, and read after Do returns, which is what keeps the
		// race detector out of this.
		received := make(chan string, 1)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received <- r.Header.Get("Traceparent")

			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		// Extracting an inbound traceparent puts a valid span context on ctx without pulling in the
		// tracing SDK: the outbound request should continue that trace rather than start its own.
		carrier := propagation.HeaderCarrier{}
		carrier.Set("traceparent", inboundTraceHeader)
		ctx := propagation.TraceContext{}.Extract(t.Context(), carrier)

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		response, err := lib.NewHTTPClient(lib.HTTPClientOptions{}).Do(request)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())

		require.Contains(t, <-received, traceID)
	})
}
