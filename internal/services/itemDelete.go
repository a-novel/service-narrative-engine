package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

type ItemDeleteRepository interface {
	Exec(ctx context.Context, request *dao.ItemDeleteRequest) (*dao.Item, error)
}

type ItemDeleteRequest struct {
	// ID identifies the item to remove; must be a non-zero UUID. `uuid.Nil`
	// (the all-zero UUID) is almost always a missing path/query parameter
	// rather than a real lookup.
	ID uuid.UUID `validate:"required"`
}

// ItemDelete removes an item by its ID.
type ItemDelete struct {
	repository ItemDeleteRepository
}

func NewItemDelete(repository ItemDeleteRepository) *ItemDelete {
	return &ItemDelete{repository: repository}
}

func (service *ItemDelete) Exec(ctx context.Context, request *ItemDeleteRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ItemDelete")
	defer span.End()

	span.SetAttributes(attribute.String("item.id", request.ID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.repository.Exec(ctx, &dao.ItemDeleteRequest{ID: request.ID})
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
