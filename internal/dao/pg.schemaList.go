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

//go:embed pg.schemaList.sql
var schemaListQuery string

type SchemaListRequest struct {
	ProjectID uuid.UUID
}

type SchemaList struct{}

func NewSchemaList() *SchemaList {
	return new(SchemaList)
}

func (repository *SchemaList) Exec(ctx context.Context, request *SchemaListRequest) ([]*Schema, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SchemaList")
	defer span.End()

	span.SetAttributes(attribute.String("project_id", request.ProjectID.String()))

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var schemas []*Schema

	err = tx.NewRaw(schemaListQuery, request.ProjectID).Scan(ctx, &schemas)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, schemas), nil
}
