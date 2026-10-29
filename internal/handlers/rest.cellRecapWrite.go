package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestCellRecapWriteService is the core operation [RestCellRecapWrite] delegates to.
type RestCellRecapWriteService interface {
	Exec(ctx context.Context, request *core.CellRecapWriteRequest) error
}

// RestCellRecapWriteRequest is the JSON body the write-cell-recap endpoint accepts.
type RestCellRecapWriteRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID `json:"sceneID"`
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID `json:"layerID"`

	// Recap is the output of the consolidation that recapped the cell.
	Recap json.RawMessage `json:"recap"`
	// CellUpdatedAt is the cell's updatedAt the consolidation read, as its context reports it.
	CellUpdatedAt time.Time `json:"cellUpdatedAt"`
}

// RestCellRecapWrite is the HTTP handler for the write-cell-recap endpoint. It responds 204 once the recap
// is stored, and 409 when the recap read the cell at a version older than the stored recap's, or newer
// than the cell's own.
type RestCellRecapWrite struct {
	service RestCellRecapWriteService
	logger  logging.Log
}

func NewRestCellRecapWrite(service RestCellRecapWriteService, logger logging.Log) *RestCellRecapWrite {
	return &RestCellRecapWrite{service: service, logger: logger}
}

func (handler *RestCellRecapWrite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CellRecapWrite")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestCellRecapWriteRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	err = handler.service.Exec(ctx, &core.CellRecapWriteRequest{
		OwnerID:       ownerID,
		ProjectID:     request.ProjectID,
		SceneID:       request.SceneID,
		LayerID:       request.LayerID,
		Recap:         request.Recap,
		CellUpdatedAt: request.CellUpdatedAt,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, recapErrors, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
