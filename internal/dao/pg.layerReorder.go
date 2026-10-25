package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.layerReorder.sql
var layerReorderQuery string

// LayerReorderRequest is the input to [PgLayerReorder.Exec].
type LayerReorderRequest struct {
	ProjectID uuid.UUID
	// IDs lists every layer of the project, in their new order.
	IDs []uuid.UUID
}

// PgLayerReorder sets the positions of a project's layers to the order of IDs. It returns the layers
// it moved, in no particular order.
type PgLayerReorder struct{}

func NewPgLayerReorder() *PgLayerReorder {
	return &PgLayerReorder{}
}

func (dao *PgLayerReorder) Exec(ctx context.Context, request *LayerReorderRequest) ([]*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayerReorder")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Layer

	err = db.NewRaw(layerReorderQuery, request.ProjectID, bun.List(request.IDs)).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("layers.count", len(entities)))

	return entities, nil
}
