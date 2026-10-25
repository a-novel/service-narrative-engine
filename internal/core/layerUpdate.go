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

// LayerUpdateDaoSelect reads the layer when the write missed, to tell a deleted layer from a stale
// revision.
type LayerUpdateDaoSelect interface {
	Exec(ctx context.Context, request *dao.LayerSelectRequest) (*dao.Layer, error)
}

// LayerUpdateRequest carries a layer's new fields and the revision the caller read.
type LayerUpdateRequest struct {
	OwnerID     uuid.UUID `validate:"required"`
	ID          uuid.UUID `validate:"required"`
	Revision    int       `validate:"min=1"`
	Name        string    `validate:"required,notblank,max=100"`
	Description string    `validate:"max=1000"`
}

// LayerUpdate replaces a layer's name and description. It answers [ErrRevisionConflict] when the layer
// changed since the caller read it.
type LayerUpdate struct {
	dao       LayerUpdateDao
	daoSelect LayerUpdateDaoSelect
}

func NewLayerUpdate(dao LayerUpdateDao, daoSelect LayerUpdateDaoSelect) *LayerUpdate {
	return &LayerUpdate{dao: dao, daoSelect: daoSelect}
}

func (service *LayerUpdate) Exec(ctx context.Context, request *LayerUpdateRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("layer.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("layer.revision", request.Revision),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.LayerUpdateRequest{
		ID:          request.ID,
		OwnerID:     request.OwnerID,
		Revision:    request.Revision,
		Name:        request.Name,
		Description: request.Description,
		Now:         time.Now(),
	})
	if errors.Is(err, dao.ErrLayerUpdateNotFound) {
		err = service.missed(ctx, request.ID, request.OwnerID, err)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update layer: %w", err))
	}

	return newLayer(entity, 0), nil
}

// missed tells why the write matched no row: the layer is gone, or its revision moved on.
func (service *LayerUpdate) missed(ctx context.Context, id, ownerID uuid.UUID, err error) error {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerUpdate(missed)")
	defer span.End()

	_, selectErr := service.daoSelect.Exec(ctx, &dao.LayerSelectRequest{ID: id, OwnerID: ownerID})
	if errors.Is(selectErr, dao.ErrLayerSelectNotFound) {
		return otel.ReportError(span, errors.Join(err, ErrLayerNotFound))
	}

	if selectErr != nil {
		return otel.ReportError(span, fmt.Errorf("select layer: %w", selectErr))
	}

	return otel.ReportError(span, errors.Join(err, ErrRevisionConflict))
}
