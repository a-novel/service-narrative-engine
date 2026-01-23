package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.schemaListVersions.sql
var schemaListVersionsQuery string

type SchemaListVersionsRequest struct {
	ProjectID       uuid.UUID
	ModuleID        string
	ModuleNamespace string
	Limit           int
	Offset          int
}

type SchemaVersion struct {
	ID        uuid.UUID `bun:"id"`
	CreatedAt time.Time `bun:"created_at"`
}

type SchemaListVersions struct{}

func NewSchemaListVersions() *SchemaListVersions {
	return new(SchemaListVersions)
}

func (repository *SchemaListVersions) Exec(
	ctx context.Context, request *SchemaListVersionsRequest,
) ([]*SchemaVersion, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SchemaListVersions")
	defer span.End()

	span.SetAttributes(
		attribute.String("project_id", request.ProjectID.String()),
		attribute.String("module_id", request.ModuleID),
		attribute.String("module_namespace", request.ModuleNamespace),
		attribute.Int("data.limit", request.Limit),
		attribute.Int("data.offset", request.Offset),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var versions []*SchemaVersion

	err = tx.NewRaw(
		schemaListVersionsQuery,
		request.ProjectID,
		request.ModuleID,
		request.ModuleNamespace,
		bun.NullZero(request.Limit),
		request.Offset,
	).Scan(ctx, &versions)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	if versions == nil {
		versions = []*SchemaVersion{}
	}

	return otel.ReportSuccess(span, versions), nil
}
