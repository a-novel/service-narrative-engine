package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ItemListDao is the DAO port ItemList depends on to read a page of items.
type ItemListDao interface {
	Exec(ctx context.Context, request *dao.ItemListRequest) ([]*dao.Item, error)
}

const (
	// ItemListDefaultSize is applied to ItemListRequest.Limit when the caller
	// leaves it unset (zero or negative).
	ItemListDefaultSize = 20
	// ItemListMaxSize caps the number of items returned per page. Keep the
	// `max=` constraint on ItemListRequest.Limit in sync with this value.
	ItemListMaxSize = 100
)

// ItemListRequest holds the pagination input for [ItemList.Exec].
type ItemListRequest struct {
	// Limit falls back to ItemListDefaultSize when zero or negative.
	Limit  int `validate:"max=100"`
	Offset int `validate:"min=0"`
}

// ItemList retrieves a paginated list of items.
type ItemList struct {
	dao ItemListDao
}

// NewItemList builds an ItemList that reads through the given DAO.
func NewItemList(dao ItemListDao) *ItemList {
	return &ItemList{dao: dao}
}

func (service *ItemList) Exec(ctx context.Context, request *ItemListRequest) ([]*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ItemList")
	defer span.End()

	// Resolve the page size into a local so the caller's request stays untouched.
	limit := request.Limit
	if limit <= 0 {
		limit = ItemListDefaultSize
	}

	span.SetAttributes(
		attribute.Int("item.limit", limit),
		attribute.Int("item.offset", request.Offset),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entities, err := service.dao.Exec(ctx, &dao.ItemListRequest{
		Limit:  limit,
		Offset: request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list items: %w", err))
	}

	span.SetAttributes(attribute.Int("items.count", len(entities)))

	items := make([]*Item, len(entities))
	for i, entity := range entities {
		items[i] = &Item{
			ID:          entity.ID,
			Name:        entity.Name,
			Description: entity.Description,
			CreatedAt:   entity.CreatedAt,
			UpdatedAt:   entity.UpdatedAt,
		}
	}

	return otel.ReportSuccess(span, items), nil
}
