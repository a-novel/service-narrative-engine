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

// RestGenerationCancelService is the core operation [RestGenerationCancel] delegates to.
type RestGenerationCancelService interface {
	Exec(ctx context.Context, request *core.GenerationCancelRequest) (*core.GenerationResult, error)
}

// RestGenerationCancelRequest holds the query parameters of the cancel-generation endpoint.
type RestGenerationCancelRequest struct {
	// ID names the genai generation that runs the call.
	ID uuid.UUID `schema:"id"`
}

// RestGenerationCancel is the HTTP handler for the cancel-generation endpoint. It cancels one call of a
// generation through genai, and responds with the genai generation as the cancel left it: a call that
// settled first keeps its output.
type RestGenerationCancel struct {
	service RestGenerationCancelService
	logger  logging.Log
}

func NewRestGenerationCancel(service RestGenerationCancelService, logger logging.Log) *RestGenerationCancel {
	return &RestGenerationCancel{service: service, logger: logger}
}

func (handler *RestGenerationCancel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.GenerationCancel")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestGenerationCancelRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.GenerationCancelRequest{OwnerID: ownerID, ID: request.ID})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, generationErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadGenerationResult(result))
}
