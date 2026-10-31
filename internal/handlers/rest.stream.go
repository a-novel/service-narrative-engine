package handlers

import (
	"encoding/json"
	"fmt"
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
