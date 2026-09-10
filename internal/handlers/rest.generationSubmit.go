package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

const generationRetryAfterSeconds = "5"

var errIdempotencyKeyMissing = errors.New("Idempotency-Key header is required")

// RestGenerationSubmitService submits one complete client-composed generation.
type RestGenerationSubmitService interface {
	Exec(
		ctx context.Context,
		request *core.GenerationSubmitRequest,
	) (*core.GenerationSubmitResult, error)
}

// RestGenerationSubmitRequest contains every value forwarded to generation.
type RestGenerationSubmitRequest struct {
	Instructions string          `json:"instructions"`
	Input        json.RawMessage `json:"input"`
	Context      json.RawMessage `json:"context"`
	OutputSchema json.RawMessage `json:"outputSchema"`
}

// RestGenerationSubmit publishes generic Generation submission over HTTP.
type RestGenerationSubmit struct {
	service RestGenerationSubmitService
	logger  logging.Log
}

// NewRestGenerationSubmit creates a generic Generation submit handler.
func NewRestGenerationSubmit(
	service RestGenerationSubmitService,
	logger logging.Log,
) *RestGenerationSubmit {
	return &RestGenerationSubmit{service: service, logger: logger}
}

// ServeHTTP creates or replays owner-scoped Generation work.
func (handler *RestGenerationSubmit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.GenerationSubmit")
	defer span.End()

	actor, projectID, err := restProjectIdentity(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httpf.HandleError(
			ctx,
			handler.logger,
			w,
			span,
			restTransportErrors,
			errIdempotencyKeyMissing,
		)

		return
	}

	var request RestGenerationSubmitRequest

	err = decodeRestJSON(r, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.GenerationSubmitRequest{
		Actor:          *actor,
		ProjectID:      projectID,
		IdempotencyKey: idempotencyKey,
		Instructions:   request.Instructions,
		Input:          request.Input,
		Context:        request.Context,
		OutputSchema:   request.OutputSchema,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restServiceErrors, err)

		return
	}

	w.Header().Set(
		"Location",
		"/v0/projects/"+projectID.String()+"/generations/"+result.Generation.ID.String(),
	)

	status := http.StatusOK
	if result.Created {
		status = http.StatusAccepted

		w.Header().Set("Retry-After", generationRetryAfterSeconds)
	}

	httpf.SendJSONStatus(ctx, w, span, status, loadRestGeneration(result.Generation))
}
