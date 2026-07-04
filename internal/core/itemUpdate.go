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

// ItemUpdateDao is the DAO port ItemUpdate depends on to overwrite an item's
// fields and return the stored entity.
type ItemUpdateDao interface {
	Exec(ctx context.Context, request *dao.ItemUpdateRequest) (*dao.Item, error)
}

// ItemUpdateRequest holds the validated input for [ItemUpdate.Exec]; ID selects
// the item to modify.
type ItemUpdateRequest struct {
	ID          uuid.UUID
	Name        string `validate:"required,notblank,max=256"`
	Description string `validate:"max=1024"`
}

// ItemUpdate validates and updates an existing item's fields.
type ItemUpdate struct {
	dao ItemUpdateDao
}

// NewItemUpdate builds an ItemUpdate that persists through the given DAO.
func NewItemUpdate(dao ItemUpdateDao) *ItemUpdate {
	return &ItemUpdate{dao: dao}
}

func (service *ItemUpdate) Exec(ctx context.Context, request *ItemUpdateRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ItemUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("item.id", request.ID.String()),
		attribute.String("item.name", request.Name),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.ItemUpdateRequest{
		ID:          request.ID,
		Name:        request.Name,
		Description: request.Description,
		Now:         time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update item: %w", err))
	}

	return otel.ReportSuccess(span, &Item{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}), nil
}
