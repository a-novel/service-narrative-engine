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

// ItemGetDao is the persistence dependency ItemGet uses to read an item.
type ItemGetDao interface {
	Exec(ctx context.Context, request *dao.ItemGetRequest) (*dao.Item, error)
}

// ItemGetRequest identifies the item to fetch.
type ItemGetRequest struct {
	// ID of the item. uuid.Nil is rejected as an unset identifier, usually a
	// missing request parameter.
	ID uuid.UUID `validate:"required"`
}

// ItemGet retrieves an item by its ID.
type ItemGet struct {
	dao ItemGetDao
}

func NewItemGet(dao ItemGetDao) *ItemGet {
	return &ItemGet{dao: dao}
}

func (service *ItemGet) Exec(ctx context.Context, request *ItemGetRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ItemGet")
	defer span.End()

	span.SetAttributes(attribute.String("item.id", request.ID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.ItemGetRequest{ID: request.ID})
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
