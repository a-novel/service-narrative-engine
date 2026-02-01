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

//go:embed pg.moduleListNamespaces.sql
var moduleListNamespacesQuery string

type ModuleListNamespacesRequest struct {
	Limit  int
	Offset int
}

type ModuleNamespace struct {
	Namespace string `bun:"namespace"`
}

type ModuleListNamespaces struct{}

func NewModuleListNamespaces() *ModuleListNamespaces {
	return new(ModuleListNamespaces)
}

func (repository *ModuleListNamespaces) Exec(
	ctx context.Context, request *ModuleListNamespacesRequest,
) ([]string, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ModuleListNamespaces")
	defer span.End()

	span.SetAttributes(
		attribute.Int("data.limit", request.Limit),
		attribute.Int("data.offset", request.Offset),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var namespaces []*ModuleNamespace

	err = tx.NewRaw(
		moduleListNamespacesQuery,
		bun.NullZero(request.Limit),
		request.Offset,
	).Scan(ctx, &namespaces)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	result := make([]string, len(namespaces))
	for i, ns := range namespaces {
		result[i] = ns.Namespace
	}

	return otel.ReportSuccess(span, result), nil
}
