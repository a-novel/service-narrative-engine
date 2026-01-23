package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.schemaMetaSelect.sql
var schemaMetaSelectQuery string

var ErrSchemaMetaSelectNotFound = errors.New("schema not found")

type SchemaMetaSelectRequest struct {
	// ID is the specific schema version ID. If not provided, the latest schema for the project will be retrieved.
	ID *uuid.UUID
	// ProjectID is required when ID is not provided.
	ProjectID uuid.UUID
	// ModuleID is required when ID is not provided.
	ModuleID string
	// ModuleNamespace is required when ID is not provided.
	ModuleNamespace string
}

type SchemaMetaSelect struct{}

func NewSchemaMetaSelect() *SchemaMetaSelect {
	return new(SchemaMetaSelect)
}

func (repository *SchemaMetaSelect) Exec(ctx context.Context, request *SchemaMetaSelectRequest) (*SchemaMeta, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SchemaMetaSelect")
	defer span.End()

	if request.ID != nil {
		span.SetAttributes(attribute.String("id", request.ID.String()))
	} else {
		span.SetAttributes(
			attribute.String("project_id", request.ProjectID.String()),
			attribute.String("module_id", request.ModuleID),
			attribute.String("module_namespace", request.ModuleNamespace),
		)
	}

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(SchemaMeta)

	err = tx.NewRaw(
		schemaMetaSelectQuery,
		bun.NullZero(request.ID),
		request.ProjectID,
		request.ModuleID,
		request.ModuleNamespace,
	).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrSchemaMetaSelectNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
