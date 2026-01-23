package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.schemaInsert.sql
var schemaInsertQuery string

var ErrSchemaInsertAlreadyExists = errors.New("schema already exists")

type SchemaInsertRequest struct {
	ID               uuid.UUID
	ProjectID        uuid.UUID
	Owner            *uuid.UUID
	ModuleID         string
	ModuleNamespace  string
	ModuleVersion    string
	ModulePreversion string
	Source           SchemaSource
	Data             map[string]any
	Now              time.Time
}

type SchemaInsert struct{}

func NewSchemaInsert() *SchemaInsert {
	return new(SchemaInsert)
}

func (repository *SchemaInsert) Exec(ctx context.Context, request *SchemaInsertRequest) (*Schema, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SchemaInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("id", request.ID.String()),
		attribute.String("project_id", request.ProjectID.String()),
		attribute.String("module_id", request.ModuleID),
		attribute.String("module_namespace", request.ModuleNamespace),
		attribute.String("module_version", request.ModuleVersion),
		attribute.String("module_preversion", request.ModulePreversion),
		attribute.String("source", string(request.Source)),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Schema)

	err = tx.NewRaw(
		schemaInsertQuery,
		request.ID,
		request.ProjectID,
		request.Owner,
		request.ModuleID,
		request.ModuleNamespace,
		request.ModuleVersion,
		request.ModulePreversion,
		request.Source,
		request.Data,
		request.Now,
	).Scan(ctx, entity)
	if err != nil {
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
			err = errors.Join(err, ErrSchemaInsertAlreadyExists)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
