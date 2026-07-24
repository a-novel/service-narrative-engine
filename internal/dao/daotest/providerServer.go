// Package daotest holds the fixtures data-access tests exercise an outbound provider with.
//
// It lives in regular (non-_test.go) files so tests in other packages can import it: Go excludes
// _test.go files from a package's exported surface, which is the same reasoning
// [github.com/a-novel/service-narrative-engine/internal/config/configtest] records. Production code
// never imports it, a rule enforced in review rather than by the compiler.
package daotest

import (
	"cmp"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"
	"time"
)

// ProviderResponse is one scripted reply. The zero value answers 200 with an empty body.
type ProviderResponse struct {
	// Status is the code to write. Zero writes http.StatusOK.
	Status int
	// Header is written before the status line.
	Header http.Header
	// Body is written after it.
	Body string

	// Delay holds the reply back before anything is written, which is how a test reaches a
	// caller's deadline without waiting out a real provider's latency.
	Delay time.Duration

	// Hang holds the reply open until the request's context is cancelled, and writes nothing.
	// A provider that accepts a request and then goes quiet is one of the two failures a recorded
	// fixture cannot reproduce, and it is the one a lease exists to survive.
	Hang bool

	// Drop closes the connection without writing a reply, which surfaces to the caller as a
	// transport error rather than a status code. This is the other failure a fixture cannot
	// reproduce.
	Drop bool
}

// RecordedRequest is one request the server received, captured before its reply was chosen.
type RecordedRequest struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Body   []byte
}

// ProviderServer is a scripted stand-in for an outbound HTTP provider. It replays the responses it
// was built with, one per request in order, and records everything it received.
//
// Build one with [NewProviderServer]. It shuts down with the test that made it.
type ProviderServer struct {
	t *testing.T

	server *httptest.Server

	mu        sync.Mutex
	responses []ProviderResponse
	requests  []RecordedRequest
}

// NewProviderServer starts a server that replays responses in order. A request arriving after the
// last scripted reply fails the test rather than being answered with something nobody chose.
func NewProviderServer(t *testing.T, responses ...ProviderResponse) *ProviderServer {
	t.Helper()

	provider := &ProviderServer{t: t, responses: responses}

	provider.server = httptest.NewServer(http.HandlerFunc(provider.handle))
	t.Cleanup(provider.server.Close)

	return provider
}

// URL is the base address to point a client at.
func (provider *ProviderServer) URL() string {
	return provider.server.URL
}

// Requests returns what the server received, in order.
func (provider *ProviderServer) Requests() []RecordedRequest {
	provider.mu.Lock()
	defer provider.mu.Unlock()

	return slices.Clone(provider.requests)
}

func (provider *ProviderServer) handle(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		provider.t.Errorf("daotest: read request body: %v", err)
	}

	response, scripted := provider.take(RecordedRequest{
		Method: request.Method,
		Path:   request.URL.Path,
		Query:  request.URL.Query(),
		Header: request.Header.Clone(),
		Body:   body,
	})
	if !scripted {
		// Errorf rather than Fatalf: this runs on the server's goroutine, and only Errorf is safe
		// to call from one other than the test's own.
		provider.t.Errorf("daotest: unscripted request %s %s", request.Method, request.URL.Path)
		writer.WriteHeader(http.StatusInternalServerError)

		return
	}

	if response.Drop {
		provider.drop(writer)

		return
	}

	if response.Delay > 0 && !sleep(request, response.Delay) {
		return
	}

	if response.Hang {
		<-request.Context().Done()

		return
	}

	for key, values := range response.Header {
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}

	writer.WriteHeader(cmp.Or(response.Status, http.StatusOK))

	_, err = io.WriteString(writer, response.Body)
	if err != nil {
		provider.t.Errorf("daotest: write response body: %v", err)
	}
}

func (provider *ProviderServer) drop(writer http.ResponseWriter) {
	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		provider.t.Errorf("daotest: response writer cannot be hijacked, so the connection cannot be dropped")

		return
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		provider.t.Errorf("daotest: hijack connection: %v", err)

		return
	}

	_ = conn.Close()
}

// take appends the request and pops the reply scripted for it, reporting false once the script has
// run out.
func (provider *ProviderServer) take(request RecordedRequest) (ProviderResponse, bool) {
	provider.mu.Lock()
	defer provider.mu.Unlock()

	provider.requests = append(provider.requests, request)

	if len(provider.responses) == 0 {
		return ProviderResponse{}, false
	}

	response := provider.responses[0]
	provider.responses = provider.responses[1:]

	return response, true
}

// sleep waits out duration, reporting whether it finished rather than the request going away.
func sleep(request *http.Request, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-request.Context().Done():
		return false
	case <-timer.C:
		return true
	}
}
