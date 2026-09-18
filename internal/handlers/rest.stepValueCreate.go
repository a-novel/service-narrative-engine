package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestStepValueCreateService saves one opaque Step Value.
type RestStepValueCreateService interface {
	Exec(ctx context.Context, request *core.StepValueCreateRequest) (*core.StepValue, error)
}

// RestStepValueCreateRequest contains one opaque keyed value.
type RestStepValueCreateRequest struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

// RestStepValueCreate publishes opaque Step writes over HTTP.
type RestStepValueCreate struct {
	service RestStepValueCreateService
	logger  logging.Log
}

// NewRestStepValueCreate creates a Step Value handler.
func NewRestStepValueCreate(
	service RestStepValueCreateService,
	logger logging.Log,
) *RestStepValueCreate {
	return &RestStepValueCreate{service: service, logger: logger}
}

// ServeHTTP appends one opaque Step Value under an owned Project.
func (handler *RestStepValueCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.StepValueCreate")
	defer span.End()

	actor, projectID, err := restProjectIdentity(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	var request RestStepValueCreateRequest

	err = decodeRestJSON(r, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	value, err := handler.service.Exec(ctx, &core.StepValueCreateRequest{
		Actor:     *actor,
		ProjectID: projectID,
		Key:       request.Key,
		Value:     request.Value,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restServiceErrors, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusCreated, loadRestStepValueVersion(value))
}
