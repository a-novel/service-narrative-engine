package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ItemDeleteDao is the DAO port ItemDelete depends on to remove an item and
// return the deleted entity.
type ItemDeleteDao interface {
	Exec(ctx context.Context, request *dao.ItemDeleteRequest) (*dao.Item, error)
}

// ItemDeleteRequest holds the input for [ItemDelete.Exec].
type ItemDeleteRequest struct {
	// ID selects the item to remove; the required tag rejects the zero UUID.
	ID uuid.UUID `validate:"required"`
}

// ItemDelete removes an item by its ID.
type ItemDelete struct {
	dao ItemDeleteDao
}

// NewItemDelete builds an ItemDelete that persists through the given DAO.
func NewItemDelete(dao ItemDeleteDao) *ItemDelete {
	return &ItemDelete{dao: dao}
}

func (service *ItemDelete) Exec(ctx context.Context, request *ItemDeleteRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ItemDelete")
	defer span.End()

	span.SetAttributes(attribute.String("item.id", request.ID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.ItemDeleteRequest{ID: request.ID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("delete item: %w", err))
	}

	return otel.ReportSuccess(span, &Item{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}), nil
}
