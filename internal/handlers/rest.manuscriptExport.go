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

// RestManuscriptExportService is the core operation [RestManuscriptExport] delegates to.
type RestManuscriptExportService interface {
	Exec(ctx context.Context, request *core.ManuscriptExportRequest) (*core.Manuscript, error)
}

// RestManuscriptExportRequest is the query parameters the export-manuscript endpoint accepts.
type RestManuscriptExportRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	// LayerID is the manuscript layer to export.
	LayerID uuid.UUID `schema:"layerID"`
}

// RestManuscriptExport is the HTTP handler for the export-manuscript endpoint. It responds with the text
// of one manuscript layer of a project, in manuscript format v1.
type RestManuscriptExport struct {
	service RestManuscriptExportService
	logger  logging.Log
}

func NewRestManuscriptExport(service RestManuscriptExportService, logger logging.Log) *RestManuscriptExport {
	return &RestManuscriptExport{service: service, logger: logger}
}

func (handler *RestManuscriptExport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ManuscriptExport")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestManuscriptExportRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ManuscriptExportRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		LayerID:   request.LayerID,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadManuscript(result))
}
