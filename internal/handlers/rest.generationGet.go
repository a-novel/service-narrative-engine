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

// RestGenerationGetService is the core operation [RestGenerationGet] delegates to.
type RestGenerationGetService interface {
	Exec(ctx context.Context, request *core.GenerationGetRequest) (*core.GenerationResult, error)
}

// RestGenerationGetRequest holds the query parameters of the read-generation endpoint.
type RestGenerationGetRequest struct {
	// ID names the genai generation that runs the call.
	ID uuid.UUID `schema:"id"`
}

// RestGenerationGet is the HTTP handler for the read-generation endpoint. It reads one call of a
// generation through genai, and responds with the genai generation, with its output once it succeeded.
type RestGenerationGet struct {
	service RestGenerationGetService
	logger  logging.Log
}

func NewRestGenerationGet(service RestGenerationGetService, logger logging.Log) *RestGenerationGet {
	return &RestGenerationGet{service: service, logger: logger}
}

func (handler *RestGenerationGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.GenerationGet")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestGenerationGetRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.GenerationGetRequest{OwnerID: ownerID, ID: request.ID})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, generationErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadGenerationResult(result))
}
