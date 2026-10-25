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

// ItemCreateRequest is the input to [PgItemCreate.Exec].
type ItemCreateRequest struct {
	ID          uuid.UUID
	Name        string
	Description string
	// Now is the instant stored as both created_at and updated_at.
	Now time.Time
}

// PgItemCreate inserts a new item into the database.
type PgItemCreate struct{}

func NewPgItemCreate() *PgItemCreate {
	return &PgItemCreate{}
}

func (dao *PgItemCreate) Exec(ctx context.Context, request *ItemCreateRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgItemCreate")
	defer span.End()

	span.SetAttributes(attribute.String("item.id", request.ID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Item{}

	err = db.NewRaw(itemCreateQuery, request.ID, request.Name, request.Description, request.Now).Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
