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

// RestCellWriteService is the core operation [RestCellWrite] delegates to.
type RestCellWriteService interface {
	Exec(ctx context.Context, request *core.CellWriteRequest) ([]*core.WrittenCell, error)
}

// RestCellWriteItem is one cell of a [RestCellWriteRequest].
type RestCellWriteItem struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID `json:"sceneID"`
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID `json:"layerID"`

	// Content is a JSON document that follows the content schema of the layer's kind.
	Content json.RawMessage `json:"content"`
}

// RestCellWriteRequest is the JSON body the write-cells endpoint accepts.
type RestCellWriteRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	// Cells lists the cells to write.
	Cells []RestCellWriteItem `json:"cells"`
}

// RestCellWrite is the HTTP handler for the write-cells endpoint. It responds with the written cells, in the
// order of the request, without their content.
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
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.CellWriteRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Cells: lo.Map(request.Cells, func(item RestCellWriteItem, _ int) core.CellWriteItem {
			return core.CellWriteItem{SceneID: item.SceneID, LayerID: item.LayerID, Content: item.Content}
		}),
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(cell *core.WrittenCell, _ int) WrittenCell { return loadWrittenCell(cell) })
	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}
