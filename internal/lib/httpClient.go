package lib

import (
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Connection settings that are not worth a deployment knob. They restate the standard library's
// own transport defaults, which this package cannot inherit: the transport is built field by
// field because cloning http.DefaultTransport would reach for the process-global the lint rail
// forbids.
const (
	transportDialTimeout           = 30 * time.Second
	transportDialKeepAlive         = 30 * time.Second
	transportIdleConnTimeout       = 90 * time.Second
	transportTLSHandshakeTimeout   = 10 * time.Second
	transportExpectContinueTimeout = 1 * time.Second
)

// HTTPClientOptions sizes the shared outbound HTTP client.
type HTTPClientOptions struct {
	// MaxIdleConns caps the idle connections kept across every host.
	MaxIdleConns int

	// MaxIdleConnsPerHost caps the idle connections kept for a single host, and must be at least
	// the number of jobs the worker runs at once. The standard library's default is two, and
	// http.DefaultTransport does not raise it, so every concurrent call past the second one against
	// the same provider host pays a fresh TCP connection and a fresh TLS handshake — on the
	// latency-critical path, on every call, forever.
	MaxIdleConnsPerHost int
}

// NewTransport builds the transport [NewHTTPClient] runs on. Prefer that constructor; this one is
// exported so a test can read the settings below back off the returned value.
func NewTransport(options HTTPClientOptions) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   transportDialTimeout,
			KeepAlive: transportDialKeepAlive,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          options.MaxIdleConns,
		MaxIdleConnsPerHost:   options.MaxIdleConnsPerHost,
		IdleConnTimeout:       transportIdleConnTimeout,
		TLSHandshakeTimeout:   transportTLSHandshakeTimeout,
		ExpectContinueTimeout: transportExpectContinueTimeout,

		// Zero on purpose, against the advice of every HTTP hardening guide, which recommends
		// something around twenty seconds. A non-streaming model call sends no response headers
		// until generation finishes, so any value here kills exactly the long calls this client
		// exists to carry — and only the long ones, which means it passes CI and fails in
		// production. The caller's context deadline is the only budget that can give a two-second
		// metadata call and a hundred-and-fifty-second generation different limits on one shared
		// client. Written out rather than left off so the zero reads as a decision.
		ResponseHeaderTimeout: 0,
	}
}

// NewHTTPClient builds the one client every outbound provider call shares. It is traced with
// otelhttp, so each request opens a span under the caller's and carries the trace context to the
// provider.
func NewHTTPClient(options HTTPClientOptions) *http.Client {
	return &http.Client{
		Transport: otelhttp.NewTransport(NewTransport(options)),

		// Zero for the same reason ResponseHeaderTimeout is, and more bluntly: this one bounds the
		// whole exchange, body included, so any value truncates a long generation mid-stream.
		// Deadlines come from the caller's context.
		Timeout: 0,
	}
}
