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

//go:embed pg.layerList.sql
var layerListQuery string

// LayerListRequest is the input to [PgLayerList.Exec].
type LayerListRequest struct {
	// ProjectID is the project whose layers to read.
	ProjectID uuid.UUID

	// Limit caps the layers in the page.
	Limit int
	// Offset is how many layers in order come before the page.
	Offset int
}

// LayerListRow is one layer of a page.
type LayerListRow struct {
	Layer

	// Total counts every layer of the project, beyond the page.
	Total int `bun:"total"`
}

// PgLayerList reads a page of a project's layers, in order. Each row carries the total, and a page
// past the end comes back empty.
type PgLayerList struct{}

func NewPgLayerList() *PgLayerList {
	return &PgLayerList{}
}

func (dao *PgLayerList) Exec(ctx context.Context, request *LayerListRequest) ([]*LayerListRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayerList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*LayerListRow

	err = db.NewRaw(layerListQuery, request.ProjectID, request.Limit, request.Offset).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("layers.count", len(entities)))

	return entities, nil
}
