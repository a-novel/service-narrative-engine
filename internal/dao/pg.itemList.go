package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.itemList.sql
var itemListQuery string

// ItemListRequest holds the parameters for an [ItemList.Exec] call.
type ItemListRequest struct {
	Limit  int
	Offset int
}

// An ItemList retrieves a paginated list of items ordered by creation date, newest first.
type ItemList struct{}

// NewItemList returns a new ItemList DAO.
func NewItemList() *ItemList {
	return new(ItemList)
}

func (dao *ItemList) Exec(ctx context.Context, request *ItemListRequest) ([]*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ItemList")
	defer span.End()

	span.SetAttributes(
		attribute.Int("item.limit", request.Limit),
		attribute.Int("item.offset", request.Offset),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var entities []*Item

	err = tx.NewRaw(itemListQuery, request.Limit, request.Offset).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("items.count", len(entities)))

	return otel.ReportSuccess(span, entities), nil
}
