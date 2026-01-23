package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.schemaMetaList.sql
var schemaMetaListQuery string

type SchemaMetaListRequest struct {
	ProjectID uuid.UUID
}

type SchemaMetaList struct{}

func NewSchemaMetaList() *SchemaMetaList {
	return new(SchemaMetaList)
}

func (repository *SchemaMetaList) Exec(ctx context.Context, request *SchemaMetaListRequest) ([]*SchemaMeta, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SchemaMetaList")
	defer span.End()

	span.SetAttributes(
		attribute.String("project_id", request.ProjectID.String()),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var schemas []*SchemaMeta

	err = tx.NewRaw(
		schemaMetaListQuery,
		request.ProjectID,
	).Scan(ctx, &schemas)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, schemas), nil
}
