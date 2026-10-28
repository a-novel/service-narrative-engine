package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// LayersUpdateDaoOwner checks that the user owns the project.
type LayersUpdateDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// LayersUpdateDao writes the layers' new fields, all of them or none.
type LayersUpdateDao interface {
	Exec(ctx context.Context, request *dao.LayersUpdateRequest) ([]*dao.Layer, error)
}

// LayersUpdateItem is one layer of a [LayersUpdateRequest].
type LayersUpdateItem struct {
	ID uuid.UUID `validate:"required"`

	// Name labels the layer's row in the composer.
	Name string `validate:"required,notblank,max=100"`
	// Description says what the layer records.
	Description string `validate:"max=20000"`
}

// LayersUpdateRequest carries the new fields of layers.
type LayersUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// Layers lists the layers to update, each once, at most 100.
	Layers []LayersUpdateItem `validate:"required,min=1,max=100,unique=ID,dive"`
}

// LayersUpdate replaces the fields of several layers at once, all of them or none, and returns them in the
// order of the request. [LayerUpdate] updates one.
type LayersUpdate struct {
	daoOwner LayersUpdateDaoOwner
	dao      LayersUpdateDao
}

func NewLayersUpdate(daoOwner LayersUpdateDaoOwner, dao LayersUpdateDao) *LayersUpdate {
	return &LayersUpdate{daoOwner: daoOwner, dao: dao}
}

func (service *LayersUpdate) Exec(ctx context.Context, request *LayersUpdateRequest) ([]*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayersUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("layers.count", len(request.Layers)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	updated, err := service.dao.Exec(ctx, &dao.LayersUpdateRequest{
		ProjectID: request.ProjectID,
		Layers: lo.Map(request.Layers, func(item LayersUpdateItem, _ int) dao.LayersUpdateItem {
			return dao.LayersUpdateItem{ID: item.ID, Name: item.Name, Description: item.Description}
		}),
		Now: time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update layers: %w", err))
	}

	// The update writes none of the layers when the project does not hold one of them.
	if len(updated) != len(request.Layers) {
		return nil, otel.ReportError(span, fmt.Errorf(
			"%w: %d of %d layers", ErrLayerNotFound, len(request.Layers)-len(updated), len(request.Layers),
		))
	}

	byID := lo.KeyBy(updated, func(entity *dao.Layer) uuid.UUID { return entity.ID })

	return lo.Map(request.Layers, func(item LayersUpdateItem, _ int) *Layer { return newLayer(byID[item.ID]) }), nil
}
