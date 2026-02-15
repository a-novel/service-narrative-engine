package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.moduleListIDs.sql
var moduleListIDsQuery string

type ModuleListIDsRequest struct {
	Namespace string
	Limit     int
	Offset    int
}

type ModuleID struct {
	ID string `bun:"id"`
}

type ModuleListIDs struct{}

func NewModuleListIDs() *ModuleListIDs {
	return new(ModuleListIDs)
}

func (repository *ModuleListIDs) Exec(
	ctx context.Context, request *ModuleListIDsRequest,
) ([]string, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ModuleListIDs")
	defer span.End()

	span.SetAttributes(
		attribute.String("data.namespace", request.Namespace),
		attribute.Int("data.limit", request.Limit),
		attribute.Int("data.offset", request.Offset),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var modules []*ModuleID

	err = tx.NewRaw(
		moduleListIDsQuery,
		request.Namespace,
		bun.NullZero(request.Limit),
		request.Offset,
	).Scan(ctx, &modules)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	result := make([]string, len(modules))
	for i, m := range modules {
		result[i] = m.ID
	}

	return otel.ReportSuccess(span, result), nil
}
