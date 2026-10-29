package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestBeatRecapWriteService is the core operation [RestBeatRecapWrite] delegates to.
type RestBeatRecapWriteService interface {
	Exec(ctx context.Context, request *core.BeatRecapWriteRequest) error
}

// RestBeatRecapWriteRequest is the JSON body the write-beat-recap endpoint accepts.
type RestBeatRecapWriteRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	BeatID    uuid.UUID `json:"beatID"`

	// Recap is the output of the consolidation that recapped the beat.
	Recap json.RawMessage `json:"recap"`
	// BeatUpdatedAt is the beat's updatedAt the consolidation read, as its context reports it.
	BeatUpdatedAt time.Time `json:"beatUpdatedAt"`
	// Sources lists the manuscript cell the consolidation read in each scene of the beat, as its context
	// reports them.
	Sources []*BeatRecapSource `json:"sources"`
}

// RestBeatRecapWrite is the HTTP handler for the write-beat-recap endpoint. It responds 204 once the recap
// is stored, and 409 when the recap read the beat at a version older than the stored recap's, or newer
// than the beat's own.
type RestBeatRecapWrite struct {
	service RestBeatRecapWriteService
	logger  logging.Log
}

func NewRestBeatRecapWrite(service RestBeatRecapWriteService, logger logging.Log) *RestBeatRecapWrite {
	return &RestBeatRecapWrite{service: service, logger: logger}
}

func (handler *RestBeatRecapWrite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.BeatRecapWrite")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestBeatRecapWriteRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	err = handler.service.Exec(ctx, &core.BeatRecapWriteRequest{
		OwnerID:       ownerID,
		ProjectID:     request.ProjectID,
		BeatID:        request.BeatID,
		Recap:         request.Recap,
		BeatUpdatedAt: request.BeatUpdatedAt,
		Sources: lo.Map(request.Sources, func(source *BeatRecapSource, _ int) *core.BeatRecapSource {
			return saveBeatRecapSource(source)
		}),
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, recapErrors, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// saveBeatRecapSource returns nil for a missing source, which the core refuses.
func saveBeatRecapSource(source *BeatRecapSource) *core.BeatRecapSource {
	if source == nil {
		return nil
	}

	return &core.BeatRecapSource{SceneID: source.SceneID, CellUpdatedAt: lo.FromPtr(source.CellUpdatedAt)}
}
