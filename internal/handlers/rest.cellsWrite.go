package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestCellsWriteService is the core operation [RestCellsWrite] delegates to.
type RestCellsWriteService interface {
	Exec(ctx context.Context, request *core.CellsWriteRequest) ([]*core.WrittenCell, error)
}

// RestCellsWriteItem is one cell of a [RestCellsWriteRequest].
type RestCellsWriteItem struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID `json:"sceneID"`
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID `json:"layerID"`

	// Content is a JSON document that follows the content schema of the layer's kind.
	Content json.RawMessage `json:"content"`
}

// RestCellsWriteRequest is the JSON body the write-cells endpoint accepts.
type RestCellsWriteRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Cells lists the cells to write.
	Cells []RestCellsWriteItem `json:"cells"`
}

// RestCellsWrite is the HTTP handler for the write-cells endpoint. It responds with the written cells, in the
// order of the request, without their content.
type RestCellsWrite struct {
	service RestCellsWriteService
	logger  logging.Log
}

func NewRestCellsWrite(service RestCellsWriteService, logger logging.Log) *RestCellsWrite {
	return &RestCellsWrite{service: service, logger: logger}
}

func (handler *RestCellsWrite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CellsWrite")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestCellsWriteRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.CellsWriteRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Cells: lo.Map(request.Cells, func(item RestCellsWriteItem, _ int) core.CellsWriteItem {
			return loadCellsWriteItem(item)
		}),
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(cell *core.WrittenCell, _ int) WrittenCell { return loadWrittenCell(cell) })
	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}

// loadCellsWriteItem converts one item of the request body to its core counterpart.
func loadCellsWriteItem(item RestCellsWriteItem) core.CellsWriteItem {
	return core.CellsWriteItem{SceneID: item.SceneID, LayerID: item.LayerID, Content: item.Content}
}
