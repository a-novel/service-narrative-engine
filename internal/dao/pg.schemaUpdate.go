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

//go:embed pg.schemaUpdate.sql
var schemaUpdateQuery string

var ErrSchemaUpdateNotFound = errors.New("schema not found")

type SchemaUpdateRequest struct {
	ID   uuid.UUID
	Data map[string]any
	Now  time.Time
}

type SchemaUpdate struct{}

func NewSchemaUpdate() *SchemaUpdate {
	return new(SchemaUpdate)
}

func (repository *SchemaUpdate) Exec(ctx context.Context, request *SchemaUpdateRequest) (*Schema, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SchemaUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("id", request.ID.String()),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Schema)

	err = tx.NewRaw(
		schemaUpdateQuery,
		request.ID,
		request.Data,
		request.Now,
	).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrSchemaUpdateNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
