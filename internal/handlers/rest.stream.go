package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"time"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
)

// streamHeartbeat is how often an idle stream sends a comment, so proxies keep it open.
const streamHeartbeat = 15 * time.Second

// lastEventIDHeader names the header a client resumes a stream with: the ID of the last event it
// received.
const lastEventIDHeader = "Last-Event-ID"

// streamEvents answers with a server-sent event stream that sends each item of a sequence as an event. Event
// returns an item's event ID, type and content. The stream ends with the sequence, at its first error, or at
// the first send that fails.
func streamEvents[Item any](
	ctx context.Context, w http.ResponseWriter, logger logging.Log, items iter.Seq2[Item, error],
	event func(item Item) (string, string, any),
) {
	ctx, span := otel.Tracer().Start(ctx, "rest.streamEvents")
	defer span.End()

	stream, err := httpf.NewEventStream(ctx, w, streamHeartbeat)
	if err != nil {
		logger.Err(ctx, otel.ReportError(span, err).Error())

		return
	}
	defer stream.Close()

	for item, err := range items {
		// The client resumes a cut stream after the last event it received. A client that left cut it.
		if err != nil {
			if ctx.Err() == nil {
				logger.Err(ctx, err.Error())
			}

			return
		}

		id, eventType, data := event(item)

		err = sendEvent(stream, id, eventType, data)
		// A write fails once the client left.
		if err != nil {
			return
		}
	}
}

// sendEvent sends data as JSON in an event of the stream.
func sendEvent(stream *httpf.EventStream, id, eventType string, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode %s event: %w", eventType, err)
	}

	return stream.Send(httpf.Event{ID: id, Type: eventType, Data: body})
}
