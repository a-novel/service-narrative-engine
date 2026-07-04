package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.itemDelete.sql
var itemDeleteQuery string

// ErrItemDeleteNotFound is returned when no item matches the delete request.
var ErrItemDeleteNotFound = errors.New("item not found")

// ItemDeleteRequest holds the parameters for an [ItemDelete.Exec] call.
type ItemDeleteRequest struct {
	ID uuid.UUID
}

// An ItemDelete permanently removes an item by its ID.
type ItemDelete struct{}

// NewItemDelete returns a new ItemDelete DAO.
func NewItemDelete() *ItemDelete {
	return new(ItemDelete)
}

func (dao *ItemDelete) Exec(ctx context.Context, request *ItemDeleteRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ItemDelete")
	defer span.End()

	span.SetAttributes(attribute.String("item.id", request.ID.String()))

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Item)

	err = tx.NewRaw(itemDeleteQuery, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrItemDeleteNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
