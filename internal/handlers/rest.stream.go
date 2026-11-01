package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/a-novel-kit/golib/httpf"
)

// streamHeartbeat is how often an idle stream sends a comment, so proxies keep it open.
const streamHeartbeat = 15 * time.Second

// lastEventIDHeader names the header a client resumes a stream with: the ID of the last event it
// received.
const lastEventIDHeader = "Last-Event-ID"

// sendEvent sends data as JSON in an event of the stream.
func sendEvent(stream *httpf.EventStream, id, eventType string, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode %s event: %w", eventType, err)
	}

	return stream.Send(httpf.Event{ID: id, Type: eventType, Data: body})
}

// StreamCallerKey keys the cap on open streams by the caller, so one user cannot hold every
// connection. Callers without a user share a key, and the stream routes refuse them anyway.
func StreamCallerKey(r *http.Request) (string, error) {
	ownerID, err := requestOwner(r)
	if err != nil {
		return "anonymous", nil //nolint:nilerr // The route refuses the caller itself.
	}

	return ownerID.String(), nil
}
