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

//go:embed pg.itemGet.sql
var itemGetQuery string

// ErrItemGetNotFound is returned when no item matches the requested ID.
var ErrItemGetNotFound = errors.New("item not found")

// ItemGetRequest holds the parameters for an [ItemGet.Exec] call.
type ItemGetRequest struct {
	ID uuid.UUID
}

// An ItemGet retrieves an item by its ID.
type ItemGet struct{}

// NewItemGet returns a new ItemGet DAO.
func NewItemGet() *ItemGet {
	return new(ItemGet)
}

func (dao *ItemGet) Exec(ctx context.Context, request *ItemGetRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ItemGet")
	defer span.End()

	span.SetAttributes(attribute.String("item.id", request.ID.String()))

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Item)

	err = tx.NewRaw(itemGetQuery, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrItemGetNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
