package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestStepValueHistoryService reads retained values for one opaque Project key.
type RestStepValueHistoryService interface {
	Exec(ctx context.Context, request *core.StepValueHistoryRequest) ([]*core.StepValue, error)
}

// RestStepValueHistory publishes bounded Step Value history over HTTP.
type RestStepValueHistory struct {
	service RestStepValueHistoryService
	logger  logging.Log
}

// NewRestStepValueHistory creates a Step Value history handler.
func NewRestStepValueHistory(
	service RestStepValueHistoryService,
	logger logging.Log,
) *RestStepValueHistory {
	return &RestStepValueHistory{service: service, logger: logger}
}

// ServeHTTP returns retained Step Values newest first for the requested key.
func (handler *RestStepValueHistory) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.StepValueHistory")
	defer span.End()

	actor, projectID, err := restProjectIdentity(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	values, err := handler.service.Exec(ctx, &core.StepValueHistoryRequest{
		Actor:     *actor,
		ProjectID: projectID,
		Key:       r.URL.Query().Get("key"),
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restServiceErrors, err)

		return
	}

	response := make([]*RestStepValueVersion, 0, len(values))
	for _, value := range values {
		response = append(response, loadRestStepValueVersion(value))
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, response)
}
