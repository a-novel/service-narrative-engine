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

type ItemGetRepository interface {
	Exec(ctx context.Context, request *dao.ItemGetRequest) (*dao.Item, error)
}

type ItemGetRequest struct {
	// ID identifies the item to fetch; must be a non-zero UUID. `uuid.Nil`
	// (the all-zero UUID) is almost always a missing path/query parameter
	// rather than a real lookup.
	ID uuid.UUID `validate:"required"`
}

// ItemGet retrieves an item by its ID.
type ItemGet struct {
	repository ItemGetRepository
}

func NewItemGet(repository ItemGetRepository) *ItemGet {
	return &ItemGet{repository: repository}
}

func (service *ItemGet) Exec(ctx context.Context, request *ItemGetRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ItemGet")
	defer span.End()

	span.SetAttributes(attribute.String("item.id", request.ID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.repository.Exec(ctx, &dao.ItemGetRequest{ID: request.ID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get item: %w", err))
	}

	return otel.ReportSuccess(span, &Item{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}), nil
}
