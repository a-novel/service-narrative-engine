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

// ItemGetDao is the DAO port ItemGet depends on to fetch a single item by its ID.
type ItemGetDao interface {
	Exec(ctx context.Context, request *dao.ItemGetRequest) (*dao.Item, error)
}

// ItemGetRequest holds the input for [ItemGet.Exec].
type ItemGetRequest struct {
	// ID selects the item to fetch; the required tag rejects the zero UUID.
	ID uuid.UUID `validate:"required"`
}

// ItemGet retrieves an item by its ID.
type ItemGet struct {
	dao ItemGetDao
}

// NewItemGet builds an ItemGet that reads through the given DAO.
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
