package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.moduleListVersions.sql
var moduleListVersionsQuery string

type ModuleListVersionsRequest struct {
	ID        string
	Namespace string
	Limit     int
	Offset    int
	// Version filters results to only include modules with this specific version number.
	// When empty, all versions are returned.
	Version string
	// Preversion indicates whether to include preversions in the results.
	// By default, only stable versions (empty preversion) are returned.
	Preversion bool
}

type ModuleVersion struct {
	Version    string    `bun:"version"`
	Preversion string    `bun:"preversion"`
	CreatedAt  time.Time `bun:"created_at"`
}

type ModuleListVersions struct{}

func NewModuleListVersions() *ModuleListVersions {
	return new(ModuleListVersions)
}

func (repository *ModuleListVersions) Exec(
	ctx context.Context, request *ModuleListVersionsRequest,
) ([]*ModuleVersion, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ModuleListVersions")
	defer span.End()

	span.SetAttributes(
		attribute.String("id", request.ID),
		attribute.String("namespace", request.Namespace),
		attribute.Int("data.limit", request.Limit),
		attribute.Int("data.offset", request.Offset),
		attribute.String("data.version", request.Version),
		attribute.Bool("data.preversion", request.Preversion),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var versions []*ModuleVersion

	err = tx.NewRaw(
		moduleListVersionsQuery,
		request.ID,
		request.Namespace,
		bun.NullZero(request.Limit),
		request.Offset,
		request.Preversion,
		request.Version,
	).Scan(ctx, &versions)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	if versions == nil {
		versions = []*ModuleVersion{}
	}

	return otel.ReportSuccess(span, versions), nil
}
