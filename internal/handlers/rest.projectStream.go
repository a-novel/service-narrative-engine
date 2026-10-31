package handlers

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// errProjectStreamPosition is returned for a Last-Event-ID that names no position of a project stream.
var errProjectStreamPosition = errors.New("invalid project stream position")

// RestProjectStreamService is the core operation [RestProjectStream] delegates to.
type RestProjectStreamService interface {
	Exec(ctx context.Context, request *core.ProjectStreamRequest) (iter.Seq2[*core.ProjectRecord, error], error)
}

// RestProjectStreamRequest holds the query parameters of the stream-project endpoint.
type RestProjectStreamRequest struct {
	ID uuid.UUID `schema:"id"`
}

// RestProjectStream is the HTTP handler for the stream-project endpoint. It streams a project of the
// caller whole, as server-sent events: a project event, then a beat, scene, layer and cell event for
// each of them, as the list endpoints return them, then an end event.
//
// Each event's ID is its position: "project:1", then the section and how many of its records came so
// far, such as "scenes:41", then "end". A client resumes after it through the Last-Event-ID header.
// A client that resumes after the end answers 204 No Content, which tells it to stop reconnecting.
type RestProjectStream struct {
	service RestProjectStreamService
	logger  logging.Log
}

func NewRestProjectStream(service RestProjectStreamService, logger logging.Log) *RestProjectStream {
	return &RestProjectStream{service: service, logger: logger}
}

func (handler *RestProjectStream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectStream")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectStreamRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	after, err := parsePosition(r.Header.Get(lastEventIDHeader))
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	records, err := handler.service.Exec(ctx, &core.ProjectStreamRequest{
		OwnerID: ownerID, ProjectID: request.ID, After: after,
	})
	if errors.Is(err, core.ErrStreamEnded) {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	stream, err := httpf.NewEventStream(ctx, w, streamHeartbeat)
	if err != nil {
		handler.logger.Err(ctx, otel.ReportError(span, err).Error())

		return
	}
	defer stream.Close()

	for record, err := range records {
		// The client resumes a cut stream after the last event it received. A client that left cut it.
		if err != nil {
			if ctx.Err() == nil {
				handler.logger.Err(ctx, err.Error())
			}

			return
		}

		eventType, data := loadProjectRecord(record)

		err = sendEvent(stream, formatPosition(record.Position), eventType, data)
		// A write fails once the client left.
		if err != nil {
			return
		}
	}
}

// loadProjectRecord returns the event type and content of a record.
func loadProjectRecord(record *core.ProjectRecord) (string, any) {
	switch {
	case record.Project != nil:
		return "project", loadProject(record.Project)
	case record.Beat != nil:
		return "beat", loadBeat(record.Beat)
	case record.Scene != nil:
		return "scene", loadScene(record.Scene)
	case record.Layer != nil:
		return "layer", loadLayer(record.Layer)
	case record.Cell != nil:
		return "cell", loadCell(record.Cell)
	default:
		return "end", struct{}{}
	}
}

// formatPosition writes a position as an event ID.
func formatPosition(position core.ProjectStreamPosition) string {
	if position.Section == core.ProjectStreamEnd {
		return position.Section
	}

	return position.Section + ":" + strconv.Itoa(position.Count)
}

// parsePosition reads the position a client resumes a project stream after. An empty value streams from
// the project.
func parsePosition(value string) (*core.ProjectStreamPosition, error) {
	if value == "" {
		return nil, nil //nolint:nilnil // No position is no error.
	}

	if value == core.ProjectStreamEnd {
		return &core.ProjectStreamPosition{Section: core.ProjectStreamEnd}, nil
	}

	section, count, ok := strings.Cut(value, ":")
	if !ok {
		return nil, errors.Join(fmt.Errorf("%w: %q", errProjectStreamPosition, value), core.ErrInvalidRequest)
	}

	parsed, err := strconv.Atoi(count)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("%w: %w", errProjectStreamPosition, err), core.ErrInvalidRequest)
	}

	return &core.ProjectStreamPosition{Section: section, Count: parsed}, nil
}
