package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.itemCreate.sql
var itemCreateQuery string

// ItemCreateRequest holds the parameters for an [ItemCreate.Exec] call.
type ItemCreateRequest struct {
	ID          uuid.UUID
	Name        string
	Description string
	// Now is the timestamp recorded as the item's creation time.
	Now time.Time
}

// An ItemCreate inserts a new item into the database.
type ItemCreate struct{}

// NewItemCreate returns a new ItemCreate DAO.
func NewItemCreate() *ItemCreate {
	return new(ItemCreate)
}

func (dao *ItemCreate) Exec(ctx context.Context, request *ItemCreateRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ItemCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("item.id", request.ID.String()),
		attribute.String("item.name", request.Name),
		attribute.Int64("item.created_at", request.Now.Unix()),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Item)

	err = tx.NewRaw(itemCreateQuery, request.ID, request.Name, request.Description, request.Now).Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
