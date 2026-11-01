package handlers

import (
	"context"
	"errors"
	"iter"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestGenerationStreamService is the core operation [RestGenerationStream] delegates to.
type RestGenerationStreamService interface {
	Exec(
		ctx context.Context, request *core.GenerationStreamRequest,
	) (iter.Seq2[*core.GenerationEvent, error], error)
}

// RestGenerationStreamRequest holds the query parameters of the stream-generation endpoint.
type RestGenerationStreamRequest struct {
	// ID names the genai generation that runs the call.
	ID uuid.UUID `schema:"id"`
}

// RestGenerationDelta is the content of a delta event: text the model appended to its output.
type RestGenerationDelta struct {
	Text string `json:"text"`
}

// RestGenerationStream is the HTTP handler for the stream-generation endpoint. It streams one call of
// a generation as server-sent events: a delta event for each piece of text the model writes, then a
// result event, the call as the read-generation endpoint returns it. Each event's ID is its sequence
// number, which a client resumes after through the Last-Event-ID header.
//
// A call the endpoint cannot stream answers 303 See Other, to the read-generation endpoint. A client
// that resumes after the result answers 204 No Content, which tells it to stop reconnecting.
type RestGenerationStream struct {
	service RestGenerationStreamService
	logger  logging.Log
}

func NewRestGenerationStream(service RestGenerationStreamService, logger logging.Log) *RestGenerationStream {
	return &RestGenerationStream{service: service, logger: logger}
}

func (handler *RestGenerationStream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.GenerationStream")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestGenerationStreamRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	after, err := parseSequence(r.Header.Get(lastEventIDHeader))
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	events, err := handler.service.Exec(ctx, &core.GenerationStreamRequest{
		OwnerID: ownerID, ID: request.ID, After: after,
	})

	switch {
	case errors.Is(err, core.ErrGenerationNotStreamed):
		http.Redirect(w, r, "/v1/generation?id="+request.ID.String(), http.StatusSeeOther)

		return
	case errors.Is(err, core.ErrStreamEnded):
		w.WriteHeader(http.StatusNoContent)

		return
	case err != nil:
		httpf.HandleError(ctx, handler.logger, w, span, generationErrors, err)

		return
	}

	stream, err := httpf.NewEventStream(ctx, w, streamHeartbeat)
	if err != nil {
		handler.logger.Err(ctx, otel.ReportError(span, err).Error())

		return
	}
	defer stream.Close()

	for event, err := range events {
		// The client resumes a cut stream after the last event it received. A client that left cut it.
		if err != nil {
			if ctx.Err() == nil {
				handler.logger.Err(ctx, err.Error())
			}

			return
		}

		id := strconv.FormatInt(event.Sequence, 10)

		if event.Result != nil {
			err = sendEvent(stream, id, "result", loadGenerationResult(event.Result))
		} else {
			err = sendEvent(stream, id, "delta", RestGenerationDelta{Text: event.Delta})
		}

		// A write fails once the client left.
		if err != nil {
			return
		}
	}
}

// parseSequence reads the sequence number a client resumes a generation stream after. An empty value
// streams from the first event.
func parseSequence(value string) (*int64, error) {
	if value == "" {
		return nil, nil //nolint:nilnil // No position is no error.
	}

	sequence, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, errors.Join(err, core.ErrInvalidRequest)
	}

	return &sequence, nil
}
