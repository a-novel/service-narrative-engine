package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestCellWriteService is the batch operation [RestCellWrite] delegates to, with a list of one.
type RestCellWriteService interface {
	Exec(ctx context.Context, request *core.CellsWriteRequest) ([]*core.WrittenCell, error)
}

// RestCellWriteRequest is the JSON body the write-cell endpoint accepts.
type RestCellWriteRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID `json:"sceneID"`
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID `json:"layerID"`

	// Content is a JSON document that follows the content schema of the layer's kind.
	Content json.RawMessage `json:"content"`
}

// RestCellWrite is the HTTP handler for the write-cell endpoint. It responds with the written cell,
// without its content.
type RestCellWrite struct {
	service RestCellWriteService
	logger  logging.Log
}

func NewRestCellWrite(service RestCellWriteService, logger logging.Log) *RestCellWrite {
	return &RestCellWrite{service: service, logger: logger}
}

func (handler *RestCellWrite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CellWrite")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestCellWriteRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	span.SetAttributes(
		attribute.String("cell.scene_id", request.SceneID.String()),
		attribute.String("cell.layer_id", request.LayerID.String()),
		attribute.Int("cell.size", len(request.Content)),
	)

	serviceRequest := &core.CellsWriteRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Cells: []core.CellsWriteItem{{
			SceneID: request.SceneID,
			LayerID: request.LayerID,
			Content: request.Content,
		}},
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, tagLimit(itemFields(err)))

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadWrittenCell(result[0]))
}
