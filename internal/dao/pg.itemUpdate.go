package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.itemUpdate.sql
var itemUpdateQuery string

// ErrItemUpdateNotFound is returned by [PgItemUpdate.Exec] when no item has the requested ID.
var ErrItemUpdateNotFound = errors.New("item not found")

// ItemUpdateRequest is the input to [PgItemUpdate.Exec].
type ItemUpdateRequest struct {
	ID          uuid.UUID
	Name        string
	Description string
	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgItemUpdate modifies the name and description of an existing item.
type PgItemUpdate struct{}

func NewPgItemUpdate() *PgItemUpdate {
	return &PgItemUpdate{}
}

func (dao *PgItemUpdate) Exec(ctx context.Context, request *ItemUpdateRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgItemUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Item{}

	err = db.NewRaw(itemUpdateQuery, request.Name, request.Description, request.Now, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrItemUpdateNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
