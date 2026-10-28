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
}

// PgLayerList reads the layers of a project in order.
type PgLayerList struct{}

func NewPgLayerList() *PgLayerList {
	return &PgLayerList{}
}

func (dao *PgLayerList) Exec(ctx context.Context, request *LayerListRequest) ([]*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayerList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Layer

	err = db.NewRaw(layerListQuery, request.ProjectID).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("layers.count", len(entities)))

	return entities, nil
}
