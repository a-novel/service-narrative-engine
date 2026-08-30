package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestGenerationGetService reads one owner-scoped Generation.
type RestGenerationGetService interface {
	Exec(ctx context.Context, request *core.GenerationGetRequest) (*core.Generation, error)
}

// RestGenerationGet publishes Generation lifecycle reads over HTTP.
type RestGenerationGet struct {
	service RestGenerationGetService
	logger  logging.Log
}

// NewRestGenerationGet creates a Generation get handler.
func NewRestGenerationGet(service RestGenerationGetService, logger logging.Log) *RestGenerationGet {
	return &RestGenerationGet{service: service, logger: logger}
}

// ServeHTTP returns current Generation state and its opaque proposal.
func (handler *RestGenerationGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.GenerationGet")
	defer span.End()

	actor, projectID, err := restProjectIdentity(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	generationID, err := uuid.Parse(chi.URLParam(r, "generationID"))
	if err != nil {
		httpf.HandleError(
			ctx,
			handler.logger,
			w,
			span,
			restTransportErrors,
			fmt.Errorf("parse Generation ID: %w", err),
		)

		return
	}

	generation, err := handler.service.Exec(ctx, &core.GenerationGetRequest{
		Actor:     *actor,
		ProjectID: projectID,
		ID:        generationID,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restServiceErrors, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadRestGeneration(generation))
}
