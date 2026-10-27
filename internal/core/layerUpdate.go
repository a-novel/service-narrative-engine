package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// LayerUpdateDao writes the layer's new fields.
type LayerUpdateDao interface {
	Exec(ctx context.Context, request *dao.LayerUpdateRequest) (*dao.Layer, error)
}

// LayerUpdateRequest carries a layer's new fields.
type LayerUpdateRequest struct {
	OwnerID     uuid.UUID `validate:"required"`
	ID          uuid.UUID `validate:"required"`
	Name        string    `validate:"required,notblank,max=100"`
	Description string    `validate:"max=20000"`
}

// LayerUpdate replaces a layer's name and description.
type LayerUpdate struct {
	dao LayerUpdateDao
}

func NewLayerUpdate(dao LayerUpdateDao) *LayerUpdate {
	return &LayerUpdate{dao: dao}
}

func (service *LayerUpdate) Exec(ctx context.Context, request *LayerUpdateRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("layer.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.LayerUpdateRequest{
		ID:          request.ID,
		OwnerID:     request.OwnerID,
		Name:        request.Name,
		Description: request.Description,
		Now:         time.Now(),
	})
	if errors.Is(err, dao.ErrLayerUpdateNotFound) {
		err = errors.Join(err, ErrLayerNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update layer: %w", err))
	}

	return newLayer(entity, 0), nil
}
