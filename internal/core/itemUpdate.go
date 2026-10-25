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

// ItemUpdateDao is the persistence dependency ItemUpdate uses to write an item's new values.
type ItemUpdateDao interface {
	Exec(ctx context.Context, request *dao.ItemUpdateRequest) (*dao.Item, error)
}

// ItemUpdateRequest carries the target item ID and its new field values.
type ItemUpdateRequest struct {
	// ID of the item. uuid.Nil is rejected as an unset identifier, usually a
	// missing request field.
	ID          uuid.UUID `validate:"required"`
	Name        string    `validate:"required,notblank,max=256"`
	Description string    `validate:"max=1024"`
}

// ItemUpdate validates and updates an existing item's fields.
type ItemUpdate struct {
	dao ItemUpdateDao
}

func NewItemUpdate(dao ItemUpdateDao) *ItemUpdate {
	return &ItemUpdate{dao: dao}
}

func (service *ItemUpdate) Exec(ctx context.Context, request *ItemUpdateRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ItemUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("item.id", request.ID.String()),
		attribute.String("item.name", request.Name),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	daoRequest := &dao.ItemUpdateRequest{
		ID:          request.ID,
		Name:        request.Name,
		Description: request.Description,
		Now:         time.Now(),
	}

	entity, err := service.dao.Exec(ctx, daoRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update item: %w", err))
	}

	return newItem(entity), nil
}
