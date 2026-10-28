package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestCellListService is the core operation [RestCellList] delegates to.
type RestCellListService interface {
	Exec(ctx context.Context, request *core.CellListRequest) (*core.CellPage, error)
}

// RestCellListRequest is the query parameters the list-cells endpoint accepts.
type RestCellListRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`

	Limit  int `schema:"limit"`
	Offset int `schema:"offset"`
}

// RestCellList is the HTTP handler for the list-cells endpoint. It responds with a page of a
// project's cells, by the position of their scene, then of their layer, and their total.
type RestCellList struct {
	service RestCellListService
	logger  logging.Log
}

func NewRestCellList(service RestCellListService, logger logging.Log) *RestCellList {
	return &RestCellList{service: service, logger: logger}
}

func (handler *RestCellList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CellList")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestCellListRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.CellListRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Limit:     request.Limit,
		Offset:    request.Offset,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadCellPage(result))
}
