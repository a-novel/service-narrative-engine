package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.itemUpdate.sql
var itemUpdateQuery string

// ErrItemUpdateNotFound is returned when no item matches the update request.
var ErrItemUpdateNotFound = errors.New("item not found")

// ItemUpdateRequest holds the parameters for an [ItemUpdate.Exec] call.
type ItemUpdateRequest struct {
	ID          uuid.UUID
	Name        string
	Description string
	// Now is the timestamp recorded as the item's last-update time.
	Now time.Time
}

// An ItemUpdate modifies the name and description of an existing item.
type ItemUpdate struct{}

// NewItemUpdate returns a new ItemUpdate DAO.
func NewItemUpdate() *ItemUpdate {
	return new(ItemUpdate)
}

func (dao *ItemUpdate) Exec(ctx context.Context, request *ItemUpdateRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ItemUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("item.id", request.ID.String()),
		attribute.String("item.name", request.Name),
		attribute.Int64("item.updated_at", request.Now.Unix()),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Item)

	err = tx.NewRaw(itemUpdateQuery, request.Name, request.Description, request.Now, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrItemUpdateNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
