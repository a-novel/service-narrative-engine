package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.itemList.sql
var itemListQuery string

// ItemListRequest is the input to [PgItemList.Exec].
type ItemListRequest struct {
	Limit int
	// After is the ID the page starts after, in descending order. uuid.Max starts at the newest item.
	After uuid.UUID
}

// PgItemList retrieves a page of items, newest first.
type PgItemList struct{}

func NewPgItemList() *PgItemList {
	return &PgItemList{}
}

func (dao *PgItemList) Exec(ctx context.Context, request *ItemListRequest) ([]*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgItemList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Item

	err = db.NewRaw(itemListQuery, request.Limit, request.After).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("items.count", len(entities)))

	return entities, nil
}
