package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenerationGetRequest identifies one owner-scoped generation.
type GenerationGetRequest struct {
	Actor Actor     `validate:"required"`
	ID    uuid.UUID `validate:"required"`
}

// GenerationGet reads current state directly from service-genai.
type GenerationGet struct {
	engineVersionDao EngineVersionSelectDao
	genai            servicegenai.Client
}

// NewGenerationGet creates the current-state generation service.
func NewGenerationGet(
	engineVersionDao EngineVersionSelectDao,
	genai servicegenai.Client,
) *GenerationGet {
	return &GenerationGet{engineVersionDao: engineVersionDao, genai: genai}
}

// Exec returns current lifecycle state and a validated proposal on success.
func (service *GenerationGet) Exec(
	ctx context.Context,
	request *GenerationGetRequest,
) (*Generation, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationGet")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("generation.id", request.ID.String()),
		attribute.String("generation.owner_id", request.Actor.UserID.String()),
	)

	response, err := service.genai.GenerationGet(ctx, &servicegenai.GenerationGetRequest{
		Id:      request.ID.String(),
		OwnerId: request.Actor.UserID.String(),
	})
	if status.Code(err) == codes.NotFound {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %w", ErrGenerationNotFound, err))
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get generation: %w", err))
	}

	if response == nil {
		return nil, otel.ReportError(span, fmt.Errorf("%w: missing get response", ErrGenerationResponseInvalid))
	}

	generation, err := mapGeneration(
		ctx,
		service.engineVersionDao,
		response.GetGeneration(),
		&request.ID,
		request.Actor.UserID,
		nil,
	)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, generation), nil
}
