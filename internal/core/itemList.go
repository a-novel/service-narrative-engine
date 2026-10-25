package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-template/internal/dao"
)

// ItemListDao is the persistence dependency ItemList uses to read a page of items.
type ItemListDao interface {
	Exec(ctx context.Context, request *dao.ItemListRequest) ([]*dao.Item, error)
}

// ItemListDefaultSize is the page size of an [ItemListRequest] whose Limit is zero or negative.
const ItemListDefaultSize = 20

// ItemListRequest selects a page of items.
type ItemListRequest struct {
	// Limit caps the page at 100 items. It defaults to [ItemListDefaultSize] when zero or negative.
	Limit int `validate:"max=100"`
	// After is the ID of the last item of the previous page. The page starts with the next older
	// item, or with the newest one when After is uuid.Nil.
	After uuid.UUID
}

// ItemList retrieves a paginated list of items.
type ItemList struct {
	dao ItemListDao
}

func NewItemList(dao ItemListDao) *ItemList {
	return &ItemList{dao: dao}
}

func (service *ItemList) Exec(ctx context.Context, request *ItemListRequest) ([]*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ItemList")
	defer span.End()

	// Resolved locally so the caller's request stays untouched.
	limit := request.Limit
	if limit <= 0 {
		limit = ItemListDefaultSize
	}

	// Item IDs are UUIDv7, so uuid.Max sorts after every item.
	after := request.After
	if after == uuid.Nil {
		after = uuid.Max
	}

	span.SetAttributes(
		attribute.Int("item.limit", limit),
		attribute.String("item.after", request.After.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	daoRequest := &dao.ItemListRequest{
		Limit: limit,
		After: after,
	}

	entities, err := service.dao.Exec(ctx, daoRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list items: %w", err))
	}

	items := make([]*Item, len(entities))
	for i, entity := range entities {
		items[i] = newItem(entity)
	}

	return items, nil
}
