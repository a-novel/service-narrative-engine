package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.moduleExists.sql
var moduleExistsQuery string

type ModuleExists struct{}

func NewModuleExists() *ModuleExists {
	return new(ModuleExists)
}

func (repository *ModuleExists) Exec(ctx context.Context, request *ModuleSelectRequest) (bool, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ModuleExists")
	defer span.End()

	span.SetAttributes(
		attribute.String("id", request.ID),
		attribute.String("namespace", request.Namespace),
		attribute.String("version", request.Version),
		attribute.String("preversion", request.Preversion),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var exists bool

	err = tx.
		NewRaw(moduleExistsQuery, request.ID, request.Namespace, request.Version, request.Preversion).
		Scan(ctx, &exists)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, exists), nil
}
