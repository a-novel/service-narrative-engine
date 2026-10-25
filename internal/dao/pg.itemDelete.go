package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.itemDelete.sql
var itemDeleteQuery string

// ErrItemDeleteNotFound is returned by [PgItemDelete.Exec] when no item has the requested ID.
var ErrItemDeleteNotFound = errors.New("item not found")

// ItemDeleteRequest is the input to [PgItemDelete.Exec].
type ItemDeleteRequest struct {
	ID uuid.UUID
}

// PgItemDelete permanently removes an item by its ID.
type PgItemDelete struct{}

func NewPgItemDelete() *PgItemDelete {
	return &PgItemDelete{}
}

func (dao *PgItemDelete) Exec(ctx context.Context, request *ItemDeleteRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgItemDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Item{}

	err = db.NewRaw(itemDeleteQuery, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrItemDeleteNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
